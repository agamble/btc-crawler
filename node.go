package main

import (
	"encoding/gob"
	"errors"
	"github.com/btcsuite/btcd/wire"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Node struct {
	Address   string
	conn      net.Conn
	adjacents []string
	PVer      uint32
	btcNet    wire.BitcoinNet
	Online    bool

	sightings []*invSighting

	doneC   chan struct{}
	outPath string
}

type invSighting struct {
	InvVects  []wire.InvVect
	Timestamp time.Time
}

func (n *Node) Connect() error {
	conn, err := net.DialTimeout("tcp", n.Address, 30*time.Second)

	if err != nil {
		// log.Print("Connect error: ", err)
		return err
	}

	n.conn = conn
	return nil
}

func (n *Node) TcpAddress() *net.TCPAddr {
	tcpAddr, err := net.ResolveTCPAddr("tcp", n.Address)
	if err != nil {
		panic(err)
	}
	return tcpAddr
}

func (n *Node) Neighbours() []string {
	return n.adjacents
}

func (n *Node) convertNetAddress(addr *wire.NetAddress) string {
	ipString := addr.IP.String()

	if strings.Contains(ipString, ":") {
		// ipv6
		ipString = "[" + ipString + "]"
	}

	return ipString + ":" + strconv.Itoa(int(addr.Port))
}

func (n *Node) Handshake() error {
	nonce, err := wire.RandomUint64()

	if err != nil {
		log.Print("Generating nonce error:", err)
		return err
	}

	verMsg, err := wire.NewMsgVersionFromConn(n.conn, nonce, 0)

	if err != nil {
		log.Print("Create version message error:", err)
		return err
	}

	err = wire.WriteMessage(n.conn, verMsg, n.PVer, n.btcNet)

	if err != nil {
		log.Print("Write version message error:", err)
		return err
	}

	res, err := n.receiveMessageTimeout("version")

	if err != nil {
		return err
	}

	resVer, ok := res.(*wire.MsgVersion)

	if !ok {
		log.Print("Something failed getting version")
	}

	pVer := resVer.ProtocolVersion
	n.receiveMessageTimeout("verack")

	if pVer < int32(n.PVer) {
		n.PVer = uint32(pVer)
	}

	return nil
}

func (n *Node) pong(ping *wire.MsgPing) {
	pongMsg := wire.NewMsgPong(ping.Nonce)

	for i := 0; i < 2; i++ {
		err := wire.WriteMessage(n.conn, pongMsg, n.PVer, n.btcNet)

		if err != nil {
			log.Println("Failed to send pong", err)
			continue
		}

		return
	}
}

func (n *Node) ReceiveMessage(command string) (wire.Message, error) {
	for i := 0; i < 50; i++ {
		msg, _, err := wire.ReadMessage(n.conn, n.PVer, n.btcNet)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, netErr
			}

			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == syscall.ECONNRESET.Error() {
				return nil, opErr
			}

			// stop trying if there's an IO error
			if err == io.EOF || err == io.ErrUnexpectedEOF || err == io.ErrClosedPipe {
				return nil, err
			}

			// otherwise we've received some generic error, and try again
			continue
		}

		// Always respond to a ping right away
		if ping, ok := msg.(*wire.MsgPing); ok && wire.CmdPing == msg.Command() {
			n.pong(ping)
			continue
		}

		if command == msg.Command() {
			return msg, nil
		}
	}

	return nil, errors.New("Failed to receive a message from node")
}

func (n *Node) Inv(invC chan<- *invSighting) {
	res, err := n.ReceiveMessage("inv")
	now := time.Now()

	if err != nil {
		invC <- nil
		return
	}

	resInv, ok := res.(*wire.MsgInv)

	if !ok {
		invC <- nil
		return
	}

	sighting := new(invSighting)
	sighting.Timestamp = now
	sighting.InvVects = make([]wire.InvVect, len(resInv.InvList))

	for i := range resInv.InvList {
		sighting.InvVects[i] = *(resInv.InvList[i])
	}

	invC <- sighting
}

func (n *Node) Setup() error {
	err := n.Connect()

	if err != nil {
		return err
	}

	err = n.Handshake()

	if err != nil {
		return err
	}

	return nil
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (n *Node) createFiles() {
	if !exists(n.txnFilePath()) {
		os.Create(n.txnFilePath())
	}

	if !exists(n.blockFilePath()) {
		os.Create(n.blockFilePath())
	}
}

func (n *Node) txnFilePath() string {
	return n.outPath + "-txn"
}

func (n *Node) blockFilePath() string {
	return n.outPath + "-block"
}

func (n *Node) writeInv(sighting *invSighting) {
	if !exists(n.outPath) {
		os.Create(n.outPath)
	}

	f, err := os.OpenFile(n.outPath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(sighting)
	if err != nil {
		panic(err)
	}
}

func (n *Node) Watch(progressC chan<- *watchProgress, stopC chan<- string, dataDirName string) {
	n.outPath = path.Join(dataDirName, n.Address)

	resultC := make(chan *invSighting, 1)

	if err := n.Setup(); err != nil {
		return
	}

	// use a ticker to monitor watcher progress
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	go n.Inv(resultC)

	countProcessed := 0

	nilCount := 0
	for {
		select {
		case <-n.doneC:
			return
		case <-ticker.C:
			progressC <- &watchProgress{address: n.Address, uniqueInvSeen: countProcessed}
		case sighting := <-resultC:
			if sighting == nil {
				if nilCount > 5 {
					stopC <- n.Address
					return
				}
				nilCount++
			} else {
				nilCount = 0
			}

			go n.Inv(resultC)

			if sighting != nil {
				n.writeInv(sighting)
				countProcessed += len(sighting.InvVects)
			}
		}
	}
}

func (n *Node) StopWatching() {
	n.doneC <- struct{}{}
	return
}

func (n *Node) receiveMessageTimeout(command string) (wire.Message, error) {
	n.conn.SetReadDeadline(time.Time(time.Now().Add(30 * time.Second)))
	defer n.conn.SetReadDeadline(time.Time{})

	msg, err := n.ReceiveMessage(command)

	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (n *Node) GetAddr() error {
	getAddrMsg := wire.NewMsgGetAddr()
	err := wire.WriteMessage(n.conn, getAddrMsg, n.PVer, n.btcNet)

	if err != nil {
		return err
	}

	res, err := n.receiveMessageTimeout("addr")

	if err != nil {
		return err
	}

	if res == nil {
		n.adjacents = make([]string, 0)
		return nil
	}

	resAddrMsg := res.(*wire.MsgAddr)

	addrList := resAddrMsg.AddrList
	adjacents := make([]string, 0, 1000)

	for _, addr := range addrList {
		tcpAddr := n.convertNetAddress(addr)
		adjacents = append(adjacents, tcpAddr)
	}

	n.adjacents = adjacents

	return nil
}

func (n *Node) Close() error {
	if n.conn == nil {
		return nil
	}

	err := n.conn.Close()
	if err != nil {
		log.Println("Closing connection error:", err)
		return err
	}
	return nil
}

func (n *Node) IsValid() bool {
	tcpAddr := n.TcpAddress()

	// obviously a port number of zero won't work
	if tcpAddr.Port == 0 {
		return false
	}

	return true
}

func NewSeed() *Node {
	return NewNode("148.251.238.178:8333")
}

func NewNode(addr string) *Node {
	n := new(Node)
	n.Address = addr
	n.btcNet = wire.MainNet
	n.PVer = wire.ProtocolVersion
	n.doneC = make(chan struct{}, 1)

	return n
}
