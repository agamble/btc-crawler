package main

import (
	"net"
	"time"
)

type Image struct {
	// ID uint `gorm:"primary_key"`
	// Seed       *Node
	StartedAt   time.Time
	FinishedAt  time.Time
	nodes       []*Node
	seen        map[string]*Node
	onlineNodes []*Node
}

func (i *Image) OnlineNodes() []*Node {
	return i.onlineNodes
}

func (i *Image) Add(node *Node) {
	i.nodes = append(i.nodes, node)
	i.seen[node.String()] = node
}

func (i *Image) AddOnlineNode(node *Node) {
	i.onlineNodes = append(i.onlineNodes, node)
}

func (i *Image) Has(tcpAddr *net.TCPAddr) bool {
	return i.seen[tcpAddr.String()] != nil
}

func (i *Image) GetNodeFromString(nodeAddr string) *Node {
	if i.seen[nodeAddr] != nil {
		return i.seen[nodeAddr]
	}

	return nil
}

func (i *Image) GetNode(tcpAddr *net.TCPAddr) *Node {
	return i.GetNodeFromString(tcpAddr.String())
}

func NewImage() *Image {
	i := new(Image)
	// i.Seed = seed
	i.StartedAt = time.Now()
	i.nodes = make([]*Node, 0, 500000)
	i.seen = make(map[string]*Node)
	i.onlineNodes = make([]*Node, 0, 6000)
	return i
}
