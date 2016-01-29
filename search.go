package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type Neighbour struct {
	SighterId int
	SightedId int
}

func main() {
	db, _ := gorm.Open("sqlite3", "nodes.db")
	db.LogMode(true)

	db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})

	seed := NewNode("148.251.238.178:8333")
	image := NewImage(seed)
	image.Build()

	db.Create(image)

	relationships := make([]*Neighbour, 10000)

	for _, n := range image.Nodes {
		for _, neighbour := range n.Neighbours() {
			relationships = append(relationships, &Neighbour{SighterId: int(n.ID), SightedId: int(neighbour.ID)})
		}
	}

	db.Create(relationships)

}
