package main

import (
	"log"
	"time"
)

type Dispatcher struct {
	currentImage *Image
	workers      int
	jobs         chan *Node
	results      chan *Node
}

type ProgressStat struct {
	countProcessed int
	countOnline    int
	countOffline   int
	countJobs      int
}

func (d *Dispatcher) BuildImage() *Image {
	seed := NewSeed()
	image := NewImage(seed)

	d.jobs <- seed

	d.setupWorkers()
	d.processNodes(image)

	return image
}

func (d *Dispatcher) startDbWorkers() chan *Node {
	db := make(chan *Node, 1000)
	for i := 0; i < 20; i++ {
		go DbWorker(db)
	}

	return db
}

func (d *Dispatcher) processNodes(image *Image) {

	countActive := 1
	countProcessed := 0
	countOnline := 0

	// db := d.startDbWorkers()

	for {
		switch {
		case countActive > 0:
			node := <-d.results

			// db <- node

			countActive--
			countProcessed++

			if node.Online {
				countOnline++
			}

			for _, addr := range node.Neighbours() {
				if !image.Has(addr) {
					neighbour := NewNode(addr)
					image.Add(neighbour)
					d.jobs <- neighbour
					countActive++
				}
			}

			if countProcessed%100 == 0 {
				log.Println("Count online:", countOnline)
				log.Println("Count active:", countActive)
				log.Println("Jobs length:", len(d.jobs))
				log.Println("Count processed:", countProcessed)
				// log.Println("DB length:", len(db))
			}
		case countActive == 0:
			close(d.jobs)
			close(d.results)
			image.FinishedAt = time.Now()
			return
		}
	}
}

func (d *Dispatcher) setupWorkers() {
	for i := 0; i < d.workers; i++ {
		go searcher(d.jobs, d.results)
	}
}

func (d *Dispatcher) reportProgress() {
}

func NewDispatcher(workers int) *Dispatcher {
	d := new(Dispatcher)
	d.workers = workers
	d.jobs = make(chan *Node, 1000000)
	d.results = make(chan *Node, 100)

	return d
}
