package main

import (
	"log"
	"os"
	"time"
)

type Listener struct {
	status      map[string]int
	closeC      chan string
	progressC   chan *watchProgress
	onlineNodes []*Node
	duration    time.Duration
	dataDirName string
	DoneC       chan struct{}

	ListenTxs  bool
	ListenBlks bool
}

type watchProgress struct {
	uniqueInvSeen int
	address       string
}

func (l *Listener) startListeners() {
	for _, node := range l.onlineNodes {

		if l.ListenTxs {
			node.ListenTxs = true
		}

		if l.ListenBlks {
			node.ListenBlks = true
		}

		go node.Watch(l.progressC, l.closeC, l.dataDirName)
	}
}

func (l *Listener) AssertOutDirectory() {
	now := time.Now()
	l.dataDirName = "snapshot-" + now.Format(time.Stamp)

	os.Mkdir(l.dataDirName, 0777)
}

func (l *Listener) Listen() {
	l.AssertOutDirectory()
	l.startListeners()

	finished := time.After(l.duration)

	logTicker := time.NewTicker(time.Second * 5)
	defer logTicker.Stop()

	for {
		select {
		case <-finished:
			log.Println("Finished time period of listening...")
			for _, n := range l.onlineNodes {
				n.StopWatching()
			}
			log.Println("Beginning decode process...")
			time.Sleep(5)
			dec := NewDecoder(l.dataDirName)
			dec.Decode()
			l.DoneC <- struct{}{}
			return
		case <-logTicker.C:
			l.printProgress()
		case progress := <-l.progressC:
			l.status[progress.address] = progress.uniqueInvSeen
		case address := <-l.closeC:
			l.status[address] = -1
			log.Printf("Lost connection with %s", address)
		}
	}
}

func (l *Listener) printProgress() {
	sum := 0
	nodes := 0
	for _, m := range l.status {
		if m != -1 {
			sum += m
			nodes++
		}
	}
	var average int
	if nodes > 0 {
		average = sum / nodes
		log.Printf("Average inv processed: %d", average)
		log.Printf("Total inv processed: %d", sum)
	}
}

func NewListener(image *Image, duration time.Duration) *Listener {
	l := new(Listener)
	l.progressC = make(chan *watchProgress)
	l.closeC = make(chan string)
	l.DoneC = make(chan struct{})
	l.status = make(map[string]int)

	l.onlineNodes = image.OnlineNodes()
	l.duration = duration
	return l
}
