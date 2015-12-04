package main

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type Node struct {
	Address   *net.TCPAddr
	conn      net.Conn
	Adjacents []*net.TCPAddr
	PVer      uint32
	btcNet    wire.BitcoinNet
}

func (n *Node) Connect() error {
	conn, err := net.DialTimeout("tcp", n.Address.String(), 5*time.Second)

	if err != nil {
		log.Print("Connect error:", err)
		return err
	}

	if strings.Contains(":", n.Address.String()) {
		fmt.Println(n.Address.String())
	}

	n.conn = conn
	return nil
}

func (n *Node) convertNetAddress(addr wire.NetAddress) *net.TCPAddr {
	ipString := addr.IP.String()
	if strings.Contains(ipString, ":") {
		// ipv6
		ipv6Address := "[" + ipString + "]:" + strconv.Itoa(int(addr.Port))
		tcpAddr, err := net.ResolveTCPAddr("tcp6", ipv6Address)

		if err != nil {
			log.Print("Converting ipv6 address error:", err)
		}

		return tcpAddr
	} else {
		//ipv4
		concatAddress := ipString + ":" + strconv.Itoa(int(addr.Port))
		tcpAddr, err := net.ResolveTCPAddr("tcp4", concatAddress)

		if err != nil {
			log.Print("Converting address error:", err)
		}

		return tcpAddr
	}

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

	res, err := n.receiveMessage("version")

	if err != nil {
		return err
	}

	resVer, ok := res.(*wire.MsgVersion)

	if !ok {
		log.Print("Something failed getting version")
	}

	pVer := resVer.ProtocolVersion
	n.receiveMessage("verack")

	if pVer < int32(n.PVer) {
		n.PVer = uint32(pVer)
	}

	return nil
}

func (n *Node) receiveMessage(command string) (wire.Message, error) {
	count := 10
	for {
		msg, _, err := wire.ReadMessage(n.conn, n.PVer, n.btcNet)

		if err != nil {
			log.Print(err)
		}

		if msg == nil {
			if count == 0 {
				return nil, errors.New("Failed to receive response")
			}
			count--
			continue
		}

		if command == msg.Command() {
			// fmt.Println("Received message with command:", msg.Command())
			return msg, nil
		} else {
			// fmt.Println("Ignored message with command:", msg.Command())
		}
	}
}

func (n *Node) GetAddr() error {
	getAddrMsg := wire.NewMsgGetAddr()
	err := wire.WriteMessage(n.conn, getAddrMsg, n.PVer, n.btcNet)

	if err != nil {
		log.Print(err)
		return err
	}

	res, err := n.receiveMessage("addr")

	if err != nil {
		return err
	}

	if res == nil {
		n.Adjacents = make([]*net.TCPAddr, 0)
		return nil
	}

	resAddrMsg := res.(*wire.MsgAddr)

	addrList := resAddrMsg.AddrList
	tcpAddrList := make([]*net.TCPAddr, 0)

	for _, addr := range addrList {
		tcpAddrList = append(tcpAddrList, n.convertNetAddress(*addr))
	}

	n.Adjacents = tcpAddrList

	return nil
}

func (n *Node) Close() error {
	if n.conn != nil {
		err := n.conn.Close()
		if err != nil {
			log.Println("Closing connection error:", err)
			return err
		}
	}
	return nil
}

func NewNode(addr *net.TCPAddr) *Node {
	n := new(Node)
	n.Address = addr
	n.btcNet = wire.MainNet
	n.PVer = wire.ProtocolVersion

	return n
}
