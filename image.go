package main

import (
	"log"
	"time"
)

type Image struct {
	ID         int
	Seed       *Node
	StartedAt  time.Time
	FinishedAt time.Time
	Nodes      []*Node
	seen       map[string]bool
}

func (i *Image) Build() {
	jobs := make(chan *Node, 1000000)
	results := make(chan *Node, 100)
	i.seen = make(map[string]bool)

	seed := i.Seed

	jobs <- seed
	i.seen[seed.Address] = true

	i.StartedAt = time.Now()

	for i := 0; i < 4000; i++ {
		go searcher(jobs, results)
	}

	countActive := 1
	countOnline := 0
	countProcessed := 0
	for {
		switch {
		case countActive > 0:
			node := <-results
			countActive--
			countProcessed++
			if node.Online {
				countOnline++
			}
			i.Nodes = append(i.Nodes, node)
			for _, neighbour := range node.Neighbours() {
				if i.validAddress(neighbour) && !i.seen[neighbour.Address] {
					i.seen[neighbour.Address] = true
					neighbour.Image = i
					jobs <- neighbour
					countActive++
				}
			}
			if countProcessed%100 == 0 {
				log.Println("Count online:", countOnline)
				log.Println("Count active:", countActive)
				log.Println("Jobs length:", len(jobs))
				log.Println("Count processed:", countProcessed)
			}
		case countActive == 0:
			close(jobs)
			close(results)
			i.FinishedAt = time.Now()
			return
		}
	}
}

func (i *Image) Save() {
	db := DbConn()

	db.Create(i)

	for _, n := range i.Nodes {
		db.Create(n)
	}

	for _, n := range i.Nodes {
		for _, neighbour := range n.Neighbours() {
			db.Create(&Neighbour{SighterId: n.ID, SightedId: neighbour.ID})
		}
	}
}

func (i *Image) validAddress(node *Node) bool {
	tcpAddr := node.TcpAddress()

	// obviously a port number of zero won't work
	if tcpAddr.Port == 0 {
		return false
	}

	return true
}

func NewImage(seed *Node) *Image {
	i := new(Image)
	i.Seed = seed
	return i
}
