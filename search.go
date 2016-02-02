package main

import (
	"os"
	"time"
)

func setupDb() {
	// db, _ := gorm.Open("postgres", "")
	db := DbConn()

	db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})
}

func main() {

	if os.Getenv("env") == "docker-prod" {
		time.Sleep(time.Second * 10)
	}

	setupDb()

	dispatcher := NewDispatcher(4000)
	image := dispatcher.BuildImage()
	image.Save()

	// db, _ := gorm.Open("postgres", "postgres://alexander:centralised@db/btc")
	// db.LogMode(true)
	//
	// db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})

}
