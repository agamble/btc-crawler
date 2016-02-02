package main

import (
	"time"
)

type Image struct {
	ID         int
	Seed       *Node
	StartedAt  time.Time
	FinishedAt time.Time
	Nodes      []*Node
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

func (i *Image) Add(node *Node) {
	i.Nodes = append(i.Nodes, node)
}

func NewImage(seed *Node) *Image {
	i := new(Image)
	i.Seed = seed
	i.StartedAt = time.Now()
	return i
}
