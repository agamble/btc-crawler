package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	// _ "github.com/mattn/go-sqlite3"
)

func DbConn() gorm.DB {
	db, _ := gorm.Open("postgres", "username=alexander database=alexander")
	db.LogMode(true)

	return db
}
