package main

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type Node struct {
	gorm.Model

	Image     *Image
	ImageID   uint
	Address   string
	conn      net.Conn
	adjacents []string
	PVer      uint32
	btcNet    wire.BitcoinNet
	Online    bool
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

func (n *Node) receiveMessage(command string) (wire.Message, error) {
	for {
		msg, _, err := wire.ReadMessage(n.conn, n.PVer, n.btcNet)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, netErr
			}

			// log.Println("reading message error", err)
			time.Sleep(time.Second)
			continue
		}

		if msg == nil {
			time.Sleep(time.Second)
			continue
		}

		if command == msg.Command() {
			// fmt.Println("Received message with command:", msg.Command())
			return msg, nil
		}
		time.Sleep(time.Second)
		// fmt.Println("Ignored message with command:", msg.Command())
	}

	return nil, nil
}

func (n *Node) receiveMessageTimeout(command string) (wire.Message, error) {
	n.conn.SetReadDeadline(time.Time(time.Now().Add(30 * time.Second)))

	msg, err := n.receiveMessage(command)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return msg, nil
}

func (n *Node) GetAddr() error {
	getAddrMsg := wire.NewMsgGetAddr()
	err := wire.WriteMessage(n.conn, getAddrMsg, n.PVer, n.btcNet)

	if err != nil {
		log.Print(err)
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

	return n
}
