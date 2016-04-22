package main

import (
	"bufio"
	"github.com/btcsuite/btcd/wire"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	SEED_FILE = "nodes_main.txt"
)

type Crawler struct {
	progressListener chan *crawlerProgress
	jobs             chan *Node
	results          chan *jobResult
	dispatcher       bool
	workers          int
	image            *Image
	Done             chan *Image
	countActive      int

	wg *sync.WaitGroup

	// request stop through stop chan
	stop chan chan *Image
}

type crawlerProgress struct {
	countProcessed int
	countOnion     int
	countIpv6      int
	countIpv4      int
	countOnline    int
	countOffline   int
	jobs           int
	done           bool
	stopped        bool
}

type jobResult struct {
	node      *Node
	adjacents []*wire.NetAddress
}

func processNode(n *Node) ([]*wire.NetAddress, error) {
	defer n.Close()

	err := n.Connect()
	if err != nil {
		return nil, err
	}

	err = n.Handshake()
	if err != nil {
		return nil, err
	}

	// if we've managed to handshake the node is online
	n.Online = true

	adjs, err := n.GetAddr()
	if err != nil {
		return nil, err
	}

	return adjs, nil
}

func searcher(jobs <-chan *Node, results chan<- *jobResult) {
	for n := range jobs {
		adjs, err := processNode(n)

		if err != nil {
			// log.Print("Processing error failure", err)
		}

		results <- &jobResult{
			node:      n,
			adjacents: adjs,
		}
	}
}

func (c *Crawler) assertReadyToStart() bool {
	if c.workers == 0 {
		return false
	}

	if !TorUp() {
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
	log.Printf("Count Active: %d", c.countActive)
	log.Printf("Count Onion: %d", cp.countOnion)
	log.Printf("Count Ipv4: %d", cp.countIpv4)
	log.Printf("Count Ipv6: %d", cp.countIpv6)
	log.Printf("Total Online: %d", cp.countOnline)
	log.Printf("Jobs Available: %d", cp.jobs)
}

func (c *Crawler) crawl() {
	c.startWorkers()

	finished := make(chan bool)

	countProcessed := 0
	countOnline := 0
	countIpv4 := 0
	countIpv6 := 0
	countOnion := 0

	image := c.image
	stopRequested := false

	// use a ticker to monitor crawler progress
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	go func() {
		c.wg.Wait()
		finished <- true
	}()

	for {
		select {
		case <-ticker.C:
			c.printProgress(&crawlerProgress{
				countProcessed: countProcessed,
				countOnion:     countOnion,
				countIpv6:      countIpv6,
				countIpv4:      countIpv4,
				countOnline:    countOnline,
				countOffline:   countProcessed - countOnline,
				jobs:           len(c.jobs),
				done:           false,
				stopped:        stopRequested,
			})
		case result := <-c.results:
			node := result.node
			adjs := result.adjacents
			// log.Println(adjs)

			countProcessed++

			if node.Online {
				image.AddOnlineNode(node)
				countOnline++
				if node.IsTorNode() {
					countOnion++
					node.Onion = true
				} else if node.IsIpv6() {
					countIpv6++
				} else {
					countIpv4++
				}
			} else {
				image.AddOfflineNode(node)
			}

			tcpAdjs := c.processAdjacents(adjs)

			for i, addr := range tcpAdjs {
				if !image.Has(addr) {
					neighbour := NewNode(addr)
					image.Add(neighbour)

					if !stopRequested {
						c.jobs <- neighbour
						c.wg.Add(1)
						c.countActive++
					}
				}
				node.Adjacents[i] = image.GetNode(addr)
			}

			c.wg.Done()
			c.countActive--
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
				countOnion:     countOnion,
				countIpv6:      countIpv6,
				countIpv4:      countIpv4,
				countOffline:   countProcessed - countOnline,
				jobs:           len(c.jobs),
				done:           true,
				stopped:        stopRequested,
			})
			image.Ipv4Count = countIpv4
			image.Ipv6Count = countIpv6
			image.OnionCount = countOnion
			c.Done <- image
			return
		}
	}

}

func (c *Crawler) processAdjacents(adjs []*wire.NetAddress) []*net.TCPAddr {
	tcpAddrs := make([]*net.TCPAddr, 0, len(adjs))
	for _, adj := range adjs {
		tcpAddrs = append(tcpAddrs, &net.TCPAddr{
			IP:   adj.IP,
			Port: int(adj.Port),
		})
	}
	return tcpAddrs
}

func (c *Crawler) add(node *Node) {
	c.jobs <- node
	c.wg.Add(1)
	c.image.Add(node)
}

// Seed and start the crawler, should be called in its own Goroutine
func (c *Crawler) Start() {
	ready := c.assertReadyToStart()

	if !ready {
		panic("Crawler conditions are not ready to start")
	}

	c.image = NewImage()
	c.SeedCrawler()
	log.Println("Starting crawler...")
	go c.crawl()
}

// Seed the crawler manually
func (c *Crawler) SeedCrawler() error {
	log.Println("Seeding crawler...")

	file, err := os.Open(SEED_FILE)
	if err != nil {
		log.Println("Failed to open seed file...")
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		c.add(NewNodeFromString(scanner.Text()))
		c.countActive++
	}

	log.Printf("Seeded crawler with %d nodes...\n", c.countActive)
	return nil
}

// Sop will block until crawler has been killed
func (c *Crawler) Stop() {
	imageC := make(chan *Image)
	c.stop <- imageC
	<-imageC
	return
}

// Create a new crawler with a fixed number of workers
func NewCrawler(workers int) *Crawler {
	c := new(Crawler)
	c.workers = workers
	c.jobs = make(chan *Node, 1000000)
	c.results = make(chan *jobResult)
	c.Done = make(chan *Image)
	c.wg = &sync.WaitGroup{}

	return c
}
