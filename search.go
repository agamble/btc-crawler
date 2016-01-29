package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := gorm.Open("sqlite3", "nodes.db")
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&Image{}, &Node{})

	seed := NewNode("148.251.238.178:8333")
	image := NewImage(seed)
	image.Build()

	db.Create(image)

}
