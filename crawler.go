package main

import (
	"log"
	"sync"
	"time"
)

type Crawler struct {
	progressListener chan *crawlerProgress
	jobs             chan *Node
	results          chan *Node
	dispatcher       bool
	workers          int
	image            *Image
	Done             chan *Image

	// request stop through stop chan
	stop chan chan *Image
}

type crawlerProgress struct {
	countProcessed int
	countOnline    int
	countOffline   int
	jobs           int
	done           bool
	stopped        bool
}

func processNode(n *Node) error {
	defer n.Close()

	err := n.Connect()
	if err != nil {
		return err
	}

	err = n.Handshake()
	if err != nil {
		return err
	}

	err = n.GetAddr()
	if err != nil {
		return err
	}

	n.Online = true
	return nil
}

func searcher(jobs <-chan *Node, results chan<- *Node) {
	for n := range jobs {
		err := processNode(n)

		if err != nil {
			// log.Print("Processing error failure", err)
		}

		results <- n
	}
}

func (c *Crawler) assertReadyToStart() bool {
	if c.workers == 0 {
		return false
	}

	return true
}

func (c *Crawler) startWorkers() {
	for i := 0; i < c.workers; i++ {
		go searcher(c.jobs, c.results)
	}
}

func (c *Crawler) printProgress(cp *crawlerProgress) {
	log.Println("#### Crawler Progress ####")
	log.Printf("Count Processed: %d", cp.countProcessed)
	log.Printf("Count Online: %d", cp.countOnline)
	log.Printf("Jobs Available: %d", cp.jobs)
}

func (c *Crawler) crawl() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	c.startWorkers()

	finished := make(chan bool)

	countProcessed := 0
	countOnline := 0

	image := c.image
	stopRequested := false

	// use a ticker to monitor crawler progress
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	go func() {
		wg.Wait()
		finished <- true
	}()

	for {
		select {
		case <-ticker.C:
			c.printProgress(&crawlerProgress{
				countProcessed: countProcessed,
				countOnline:    countOnline,
				countOffline:   countProcessed - countOnline,
				jobs:           len(c.jobs),
				done:           false,
				stopped:        stopRequested,
			})
		case node := <-c.results:
			countProcessed++

			if node.Online {
				image.AddOnlineNode(node)
				countOnline++
			}

			for _, addr := range node.Neighbours() {
				if !image.Has(addr) {
					neighbour := NewNode(addr)
					image.Add(neighbour)

					if !stopRequested {
						c.jobs <- neighbour
						wg.Add(1)
					}
				}
			}

			wg.Done()
		case stopC := <-c.stop:
			log.Println("Crawler is slowing down...")
			stopRequested = true
			c.Done = stopC
		case <-finished:
			log.Println("Crawler is finished...")
			c.image.FinishedAt = time.Now()
			close(c.jobs)
			close(c.results)
			c.printProgress(&crawlerProgress{
				countProcessed: countProcessed,
				countOnline:    countOnline,
				countOffline:   countProcessed - countOnline,
				jobs:           len(c.jobs),
				done:           false,
				stopped:        stopRequested,
			})
			c.Done <- image
			return
		}
	}

}

func (c *Crawler) Start() {
	ready := c.assertReadyToStart()

	if !ready {
		panic("Crawler conditions are not ready to start")
	}

	seed := NewSeed()
	c.jobs <- seed
	c.image = NewImage(seed)

	log.Println("Starting crawler")

	go c.crawl()
}

// Stop will block until crawler has been killed
func (c *Crawler) Stop() {
	imageC := make(chan *Image)
	c.stop <- imageC
	<-imageC
	return
}

func NewCrawler(workers int) *Crawler {
	c := new(Crawler)
	c.workers = workers
	c.jobs = make(chan *Node, 1000000)
	c.results = make(chan *Node)
	c.Done = make(chan *Image)
	return c
}
