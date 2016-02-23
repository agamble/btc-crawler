package main

import (
	"github.com/lib/pq"
	"log"
	"time"
)

type Image struct {
	ID uint `gorm:"primary_key"`
	// Seed       *Node
	StartedAt   time.Time
	FinishedAt  time.Time
	nodes       []*Node
	seen        map[string]*Node
	onlineNodes []*Node
}

func (i *Image) Save() {
	db := DbConn()

	db.Create(i)

	i.fastSaveNodes()
	i.fastSaveNeighbours()
}

func (i *Image) OnlineNodes() []*Node {
	return i.onlineNodes
}

func (i *Image) fastSaveNodes() {
	db := DbConn()
	handle := db.DB()

	txn, err := handle.Begin()

	if err != nil {
		log.Fatal(err)
	}

	lastNode := Node{}
	db.Last(&lastNode)
	idOffset := lastNode.ID

	stmt, err := txn.Prepare(pq.CopyIn("nodes", "image_id", "address", "p_ver", "online", "created_at", "updated_at"))
	if err != nil {
		log.Fatal(err)
	}

	for index, n := range i.nodes {
		n.ID = uint(index+1) + idOffset

		_, err = stmt.Exec(i.ID, n.Address, n.PVer, n.Online, time.Now(), time.Now())
		if err != nil {
			log.Fatal(err)
		}

		if index%100 == 0 {
			log.Printf("Storing node %v\n", index)
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = txn.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func (i *Image) fastSaveNeighbours() {
	db := DbConn()
	handle := db.DB()

	txn, err := handle.Begin()

	if err != nil {
		log.Fatal(err)
	}

	stmt, err := txn.Prepare(pq.CopyIn("neighbours", "sighter_id", "sighted_id"))
	if err != nil {
		log.Fatal(err)
	}

	for index, n := range i.nodes {
		if index%100 == 0 {
			log.Printf("Storing relations for node %v\n", index)
		}
		for _, addr := range n.Neighbours() {
			neighbour := i.GetNodeFromAddr(addr)

			if n.ID == 0 || neighbour.ID == 0 {
				log.Printf("Node %v", n)
				log.Printf("Neighbour %v", neighbour)
				panic("Zero ID found")
			}
			_, err = stmt.Exec(n.ID, neighbour.ID)

			if err != nil {
				log.Fatal(err)
			}

			// relationship := Neighbour{SighterId: n.ID, SightedId: neighbour.ID}
			// db.Create(&relationship)
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = txn.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func (i *Image) Add(node *Node) {
	i.nodes = append(i.nodes, node)
	i.seen[node.Address] = node
}

func (i *Image) AddOnlineNode(node *Node) {
	i.onlineNodes = append(i.onlineNodes, node)
}

func (i *Image) Has(nodeAddr string) bool {
	return i.seen[nodeAddr] != nil
}

func (i *Image) GetNodeFromAddr(nodeAddr string) *Node {
	if i.seen[nodeAddr] != nil {
		return i.seen[nodeAddr]
	}

	return nil
}

func NewImage(seed *Node) *Image {
	i := new(Image)
	// i.Seed = seed
	i.StartedAt = time.Now()
	i.nodes = make([]*Node, 0, 500000)
	i.seen = make(map[string]*Node)
	i.onlineNodes = make([]*Node, 0, 6000)
	i.Add(seed)
	return i
}
