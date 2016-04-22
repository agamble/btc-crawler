package main

import (
	"time"
)

type Dispatcher struct {
}

// BuildImage begins main process, runs crawler and then begins listener on crawler output data
func (d *Dispatcher) BuildImage(workers int) *Image {
	crawler := NewCrawler(workers)

	crawler.Start()
	image := <-crawler.Done
	image.Save()

	listener := NewListener(image, 24*time.Hour)
	listener.ListenBlks = true

	go listener.Listen()
	<-listener.DoneC

	return image
}

func NewDispatcher() *Dispatcher {
	d := new(Dispatcher)

	return d
}
