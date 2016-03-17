package main

import (
	"encoding/json"
	"net"
	"os"
	"time"
)

type Image struct {
	StartedAt    time.Time
	FinishedAt   time.Time
	seen         map[string]*Node
	onlineNodes  []*Node
	offlineNodes []*Node
	Ipv4Count    int
	Ipv6Count    int
	OnionCount   int
}

func (i *Image) OnlineNodes() []*Node {
	return i.onlineNodes
}

func (i *Image) Add(node *Node) {
	i.seen[node.String()] = node
}

func (i *Image) AddOnlineNode(node *Node) {
	i.onlineNodes = append(i.onlineNodes, node)
}

func (i *Image) AddOfflineNode(node *Node) {
	i.offlineNodes = append(i.offlineNodes, node)
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

func (i *Image) Save() {
	f, err := os.Create("image-" + string(i.StartedAt.UnixNano()) + ".json")
	defer f.Close()

	if err != nil {
		panic(err)
		return
	}

	enc := json.NewEncoder(f)

	if err != nil {
		panic(err)
		return
	}

	enc.Encode(i)
}

func (i *Image) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Started      time.Time
		Finished     time.Time
		NodesCount   int
		OnlineCount  int
		OfflineCount int
		OnionCount   int
		IPv4Count    int
		IPv6Count    int
		OfflineNodes []*Node
		OnlineNodes  []*Node
	}{
		Started:      i.StartedAt,
		Finished:     i.FinishedAt,
		NodesCount:   len(i.onlineNodes) + len(i.offlineNodes),
		OnlineCount:  len(i.onlineNodes),
		OfflineCount: len(i.offlineNodes),
		OnionCount:   i.OnionCount,
		IPv4Count:    i.Ipv4Count,
		IPv6Count:    i.Ipv6Count,
		OfflineNodes: i.offlineNodes,
		OnlineNodes:  i.onlineNodes,
	})
}

func NewImage() *Image {
	i := new(Image)
	// i.Seed = seed
	i.StartedAt = time.Now()
	i.seen = make(map[string]*Node)
	i.offlineNodes = make([]*Node, 0, 500000)
	i.onlineNodes = make([]*Node, 0, 6000)
	return i
}
