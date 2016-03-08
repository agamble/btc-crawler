package crawler

type Neighbour struct {
	SighterId uint `gorm:"primary_key"`
	SightedId uint `gorm:"primary_key"`
}
