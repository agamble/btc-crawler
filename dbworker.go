package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	// _ "github.com/mattn/go-sqlite3"
	"os"
)

func DbConn() gorm.DB {
	var dbConnString string
	if os.Getenv("env") == "docker-prod" {
		dbConnString = "postgres://alexander:centralised@db/btc"
	} else {
		dbConnString = "username=alexander database=alexander"
	}

	db, _ := gorm.Open("postgres", dbConnString)
	db.LogMode(true)

	return db
}
