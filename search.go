package main

import (
	"github.com/btcsuite/btcd/wire"
	"os"
	"time"
)

func setupDb() {
	// db, _ := gorm.Open("postgres", "")
	db := DbConn()

	db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})
}

func startListeners(onlineNodes []*Node) {
	invPipe := make(chan []*wire.InvVect)

	go invVectHandler(invPipe)

	for _, node := range onlineNodes {
		go listener(node, invPipe)
	}
}

func main() {

	if os.Getenv("env") == "docker-prod" {
		time.Sleep(time.Second * 10)
	}

	setupDb()

	dispatcher := NewDispatcher(10000)
	image := dispatcher.BuildImage()
	image.Save()

	onlineNodes := image.OnlineNodes()
	image = nil
	startListeners(onlineNodes)

	quit := make(chan bool)

	<-quit

	return

}
