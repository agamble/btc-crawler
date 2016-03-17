package main

import (
	"encoding/gob"
	"encoding/json"
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
	TcpAddr   *net.TCPAddr
	conn      net.Conn
	Adjacents []*Node
	PVer      uint32
	btcNet    wire.BitcoinNet
	Online    bool
	Onion     bool
	Services  uint64
	UserAgent string

	doneC   chan struct{}
	outPath string

	ListenTxs  bool
	ListenBlks bool
}

type StampedInv struct {
	InvVects  []*wire.InvVect
	Timestamp time.Time
}

type StampedSighting struct {
	Timestamp time.Time
	InvVect   *wire.InvVect
}

var onioncatrange = net.IPNet{IP: net.ParseIP("FD87:d87e:eb43::"),
	Mask: net.CIDRMask(48, 128)}

func (n *Node) IsTorNode() bool {
	// bitcoind encodes a .onion address as a 16 byte number by decoding the
	// address prior to the .onion (i.e. the key hash) base32 into a ten
	// byte number. it then stores the first 6 bytes of the address as
	// 0xfD, 0x87, 0xD8, 0x7e, 0xeb, 0x43
	// this is the same range used by onioncat, part of the
	// RFC4193 Unique local IPv6 range.
	// In summary the format is:
	// { magic 6 bytes, 10 bytes base32 decode of key hash }
	return onioncatrange.Contains(n.TcpAddr.IP)
}

func (n *Node) IsIpv6() bool {
	return n.TcpAddr.IP.To4() == nil
}

func (n *Node) Connect() error {
	if n.IsTorNode() {
		// Onion Address
		conn, err := DialTor("tcp", n.TcpAddr)
		if err != nil {
			// log.Println("Tor connect error: ", err)
			return err
		}
		n.conn = conn
	} else {
		conn, err := net.DialTimeout("tcp", n.TcpAddr.String(), 30*time.Second)
		if err != nil {
			return err
		}
		n.conn = conn
	}

	return nil
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

	n.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	defer n.conn.SetWriteDeadline(time.Time{})
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

	n.PVer = uint32(resVer.ProtocolVersion)
	n.UserAgent = resVer.UserAgent
	n.Services = uint64(resVer.Services)

	n.receiveMessageTimeout("verack")

	return nil
}

func (n *Node) pong(ping *wire.MsgPing) {
	pongMsg := wire.NewMsgPong(ping.Nonce)

	for i := 0; i < 2; i++ {
		n.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		defer n.conn.SetWriteDeadline(time.Time{})

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

func (n *Node) InvWriter(dataDirName string, node <-chan *StampedInv) {
	n.outPath = path.Join(dataDirName, n.String())

	n.createFiles()

	txnFile, err := os.OpenFile(n.txnFilePath(), os.O_WRONLY|os.O_APPEND, 0666)
	defer txnFile.Close()

	if err != nil {
		panic(err)
	}

	blkFile, err := os.OpenFile(n.blockFilePath(), os.O_WRONLY|os.O_APPEND, 0666)
	defer blkFile.Close()

	if err != nil {
		panic(err)
	}

	txnEnc := gob.NewEncoder(txnFile)
	blkEnc := gob.NewEncoder(blkFile)
	stampedSightingHolder := new(StampedSighting)

	for {
		stampedInvs, ok := <-node

		if !ok {
			return
		}

		n.WriteInv(txnEnc, blkEnc, stampedSightingHolder, stampedInvs)
	}
}

func (n *Node) WriteInv(txnEnc *gob.Encoder, blkEnc *gob.Encoder, stampedSightingHolder *StampedSighting, stampedInvs *StampedInv) {
	if stampedSightingHolder == nil {
		stampedSightingHolder = new(StampedSighting)
	}

	stampedSightingHolder.Timestamp = stampedInvs.Timestamp

	for _, invVect := range stampedInvs.InvVects {
		stampedSightingHolder.InvVect = invVect
		var err error
		if n.ListenTxs && invVect.Type == wire.InvTypeTx {
			err = txnEnc.Encode(stampedSightingHolder)
		} else if n.ListenBlks && invVect.Type == wire.InvTypeBlock {
			err = blkEnc.Encode(stampedSightingHolder)
		}

		if err != nil {
			panic(err)
		}
	}
}

func (n *Node) Watch(progressC chan<- *watchProgress, stopC chan<- string, dataDirName string) {

	resultC := make(chan *StampedInv, 1)

	invWriterC := make(chan *StampedInv, 1)
	go n.InvWriter(dataDirName, invWriterC)

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
			close(invWriterC)
			return
		case <-ticker.C:
			progressC <- &watchProgress{address: n.String(), uniqueInvSeen: countProcessed}
		case stampedInv := <-resultC:
			if stampedInv == nil {
				if nilCount > 5 {
					stopC <- n.String()
					return
				}
				nilCount++
			} else {
				nilCount = 0
			}

			go n.Inv(resultC)

			if stampedInv != nil {
				invWriterC <- stampedInv
				countProcessed += len(stampedInv.InvVects)
			}
		}
	}
}

func (n *Node) Inv(invC chan<- *StampedInv) {
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

	sighting := new(StampedInv)
	sighting.Timestamp = now
	sighting.InvVects = resInv.InvList[:]

	invC <- sighting
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

func (n *Node) GetAddr() ([]*wire.NetAddress, error) {
	getAddrMsg := wire.NewMsgGetAddr()
	n.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	defer n.conn.SetWriteDeadline(time.Time{})
	err := wire.WriteMessage(n.conn, getAddrMsg, n.PVer, n.btcNet)

	if err != nil {
		return nil, err
	}

	res, err := n.receiveMessageTimeout("addr")

	if err != nil {
		return nil, err
	}

	if res == nil {
		// return empty adjacents if we receive no response
		return nil, nil
	}

	resAddrMsg := res.(*wire.MsgAddr)

	addrList := resAddrMsg.AddrList

	// allocate the memory in advance!
	n.Adjacents = make([]*Node, len(addrList))

	return addrList, nil
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

func (n *Node) String() string {
	return n.TcpAddr.String()
}

func (n *Node) IsValid() bool {

	// obviously a port number of zero won't work
	if n.TcpAddr.Port == 0 {
		return false
	}

	return true
}

func (n *Node) MarshalJSON() ([]byte, error) {
	adjsStrs := make([]string, len(n.Adjacents))

	for i, adj := range n.Adjacents {
		adjsStrs[i] = adj.String()
	}

	return json.Marshal(struct {
		Address   string
		Adjacents []string
		PVer      uint32
		Online    bool
		Onion     bool
		Services  uint64
		UserAgent string
	}{
		Address:   n.TcpAddr.String(),
		Adjacents: adjsStrs,
		PVer:      n.PVer,
		Onion:     n.Onion,
		Online:    n.Online,
		Services:  n.Services,
		UserAgent: n.UserAgent,
	})
}

func NewNodeFromString(addr string) *Node {
	if strings.Contains(addr, ".onion") {
		ip, err := OnionToIp(addr)
		if err != nil {
			panic(err)
		}
		port, err := strconv.Atoi(strings.Split(addr, ":")[1])
		if err != nil {
			panic(err)
		}
		return NewNode(&net.TCPAddr{
			IP:   ip,
			Port: port,
		})
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		panic(err)
	}

	return NewNode(tcpAddr)
}

func NewNode(tcpAddr *net.TCPAddr) *Node {
	n := new(Node)
	n.TcpAddr = tcpAddr
	n.btcNet = wire.MainNet
	n.PVer = wire.ProtocolVersion
	n.doneC = make(chan struct{}, 1)

	return n
}
