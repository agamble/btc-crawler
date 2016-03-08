package crawler

//
// import (
// 	"github.com/jinzhu/gorm"
// 	_ "github.com/lib/pq"
// 	// _ "github.com/mattn/go-sqlite3"
// 	"os"
// )
//
// func DbConn() gorm.DB {
// 	var dbConnString string
// 	if os.Getenv("env") == "docker-prod" {
// 		dbConnString = "postgres://alexander:centralised@db/btc?sslmode=disable"
// 	} else {
// 		dbConnString = "user=alexander database=alexander sslmode=disable"
// 	}
//
// 	db, _ := gorm.Open("postgres", dbConnString)
//
// 	if os.Getenv("env") != "docker-prod" {
// 		db.LogMode(true)
// 	}
//
// 	return db
// }
//
// func DbWorker(dbChan <-chan *Node) {
//
// 	dbConn := DbConn()
//
// 	for {
// 		node := <-dbChan
//
// 		err := dbConn.Create(node).Error
// 		if err != nil {
// 			panic(err)
// 		}
// 	}
//
// }
