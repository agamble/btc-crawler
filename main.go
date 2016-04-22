// This package describes the Bitcoin network crawler.
// The program's main entry point is in main.go, although the logic to dictate the running order of the crawler and listener is located in dispatcher.go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

// func setupDb() {
// 	// db, _ := gorm.Open("postgres", "")
// 	db := DbConn()
//
// 	db.AutoMigrate(&Image{}, &Node{}, &Neighbour{})
// }

func main() {
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	// if os.Getenv("env") == "docker-prod" {
	// 	time.Sleep(time.Second * 10)
	// }
	//
	// setupDb()

	dispatcher := NewDispatcher()
	_ = dispatcher.BuildImage(49000)

	return

}
