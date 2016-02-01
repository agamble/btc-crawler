package main

type Neighbour struct {
	ID        uint `gorm:"primary_key"`
	SighterId uint
	SightedId uint
}
