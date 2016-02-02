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
	seen         map[string]bool
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

	d.seen = make(map[string]bool)

	d.jobs <- seed

	d.setupWorkers()
	d.processNodes(image)

	return image
}

func (d *Dispatcher) processNodes(i *Image) {

	countActive := 1
	countProcessed := 0
	countOnline := 0

	for {
		switch {
		case countActive > 0:
			node := <-d.results
			countActive--
			countProcessed++

			if node.Online {
				countOnline++
			}

			i.Add(node)

			for _, neighbour := range node.Neighbours() {
				if neighbour.IsValid() && !d.seen[neighbour.Address] {
					d.seen[neighbour.Address] = true
					neighbour.Image = i
					d.jobs <- neighbour
					countActive++
				}
			}

			if countProcessed%100 == 0 {
				log.Println("Count online:", countOnline)
				log.Println("Count active:", countActive)
				log.Println("Jobs length:", len(d.jobs))
				log.Println("Count processed:", countProcessed)
			}
		case countActive == 0:
			close(d.jobs)
			close(d.results)
			i.FinishedAt = time.Now()
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
