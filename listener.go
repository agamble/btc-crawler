package main

import (
	"github.com/btcsuite/btcd/wire"
	"log"
	"net"
	"os"
	"time"
)

type Listener struct {
	status      map[string]int
	closeC      chan string
	addrC       chan []*wire.NetAddress
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

		go node.Watch(l.progressC, l.closeC, l.addrC, l.dataDirName)
	}
}

func (l *Listener) initialiseStatusMap() {
	for _, n := range l.onlineNodes {
		l.status[n.String()] = 0
	}
}

func (l *Listener) AssertOutDirectory() {
	now := time.Now()
	l.dataDirName = "snapshot-" + now.Format(time.Stamp)

	os.Mkdir(l.dataDirName, 0777)
}

func NetAddrToTcpAddr(netAddr *wire.NetAddress) *net.TCPAddr {
	return &net.TCPAddr{
		IP:   netAddr.IP,
		Port: int(netAddr.Port),
	}
}

func (l *Listener) Listen() {
	l.AssertOutDirectory()
	l.startListeners()

	finished := time.After(l.duration)

	logTicker := time.NewTicker(time.Second * 5)
	defer logTicker.Stop()

	for {
		select {
		case netAddrs := <-l.addrC:
			for _, netAddr := range netAddrs {
				addr := NetAddrToTcpAddr(netAddr)
				if l.status[addr.String()] == -1 {
					n := NewNode(addr)
					go n.Watch(l.progressC, l.closeC, l.addrC, l.dataDirName)
					log.Printf("Tried to start new connection to... %s", addr.String())
				}
			}
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

	log.Printf("Goroutines active for %d nodes", nodes)
}

func NewListener(image *Image, duration time.Duration) *Listener {
	l := new(Listener)
	l.progressC = make(chan *watchProgress)
	l.closeC = make(chan string)
	l.DoneC = make(chan struct{})
	l.status = make(map[string]int)
	l.addrC = make(chan []*wire.NetAddress)

	l.onlineNodes = image.OnlineNodes()
	l.duration = duration
	return l
}
