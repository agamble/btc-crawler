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

	dispatcher := NewDispatcher(4000)
	image := dispatcher.BuildImage()
	image.Save()

	// db, _ := gorm.Open("postgres", "postgres://alexander:centralised@db/btc")
	// db.LogMode(true)
	//
	// db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})

}
