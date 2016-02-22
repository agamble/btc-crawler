package main

import (
	"log"
	"time"
)

type Dispatcher struct {
	status map[string]int
}

func (d *Dispatcher) BuildImage(workers int) *Image {
	crawler := NewCrawler(workers)

	crawler.Start()
	image := <-crawler.Done

	onlineNodes := image.OnlineNodes()
	image = nil

	progressC := make(chan *watchProgress)
	doneC := make(chan string)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for _, node := range onlineNodes {
		go node.Watch(progressC, doneC)
	}

	for {
		select {
		case <-ticker.C:
			sum := 0
			for _, m := range d.status {
				if m != -1 {
					sum += m
				}
			}
			average := sum / len(onlineNodes)
			log.Printf("Average inv processed: %d", average)
			log.Printf("Total inv processed: %d", sum)
		case progress := <-progressC:
			d.status[progress.address] = progress.uniqueInvSeen
		case address := <-doneC:
			d.status[address] = -1
			log.Printf("Lost connection with %s", address)
		}
	}

	return image
}

func NewDispatcher() *Dispatcher {
	d := new(Dispatcher)
	d.status = make(map[string]int)

	return d
}
