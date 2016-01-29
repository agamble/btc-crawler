package main

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

	for i := 0; i < 200; i++ {
		go searcher(jobs, results)
	}

	for {
		select {
		case node := <-results:
			i.Nodes = append(i.Nodes, node)
			for _, neighbour := range node.Adjacents {
				if !i.seen[neighbour.Address] {
					i.seen[neighbour.Address] = true
					jobs <- neighbour
				}
			}
			fmt.Println("Jobs length:", len(jobs))
		case <-time.After(10 * time.Second):
			close(jobs)
			close(results)
			i.FinishedAt = time.Now()
		}
	}
}

func NewImage(seed *Node) *Image {
	i := new(Image)
	i.Seed = seed
	return i
}
