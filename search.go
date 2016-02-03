package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

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

	dispatcher := NewDispatcher(15000)
	image := dispatcher.BuildImage()
	image.Save()

	// db, _ := gorm.Open("postgres", "postgres://alexander:centralised@db/btc")
	// db.LogMode(true)
	//
	// db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})
	f, err := os.Create("/data/mem")
	if err != nil {
		log.Fatal(err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
	return

}
