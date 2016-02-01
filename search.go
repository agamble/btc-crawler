package main

import (
// "time"
)

func setupDb() {
	// db, _ := gorm.Open("postgres", "postgres://alexander:centralised@db/btc")
	db := DbConn()

	db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})
}

func main() {

	setupDb()

	// db, _ := gorm.Open("postgres", "postgres://alexander:centralised@db/btc")
	// db.LogMode(true)
	//
	// db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})

	seed := NewNode("148.251.238.178:8333")
	image := NewImage(seed)
	seed.Image = image

	image.Build()
	image.Save()
}
