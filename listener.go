package main

import (
	"github.com/btcsuite/btcd/wire"
	"log"
	"net"
	"os"
	"time"
)

type Listener struct {
	status      map[string]*nodeStatus
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

type nodeStatus struct {
	CountSeen int
	Active    bool
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
		l.status[n.String()] = &nodeStatus{
			Active:    false,
			CountSeen: 0,
		}
	}
}

// AssertOutDirectory creates the directory to contain the listening data
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

func (l *Listener) processNewAddrs(netAddrs []*wire.NetAddress) {
	for _, netAddr := range netAddrs {
		addrStr := NetAddrToTcpAddr(netAddr).String()

		if l.status[addrStr] == nil {
			l.status[addrStr] = &nodeStatus{
				CountSeen: 0,
				Active:    false,
			}
			log.Printf("New node discovered! %s", addrStr)
		}

		if !l.status[addrStr].Active {
			n := NewNodeFromString(addrStr)
			l.status[addrStr].Active = true
			go n.Watch(l.progressC, l.closeC, l.addrC, l.dataDirName)
			log.Printf("Tried to start new connection to... %s", addrStr)
		}
	}
}

// Listen begins listening procedure.
// Listener must be initialised first with a list of online nodes.
func (l *Listener) Listen() {
	l.AssertOutDirectory()
	l.startListeners()
	l.initialiseStatusMap()

	finished := time.After(l.duration)

	logTicker := time.NewTicker(time.Second * 5)
	defer logTicker.Stop()

	for {
		select {
		case netAddrs := <-l.addrC:
			l.processNewAddrs(netAddrs)
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
			l.status[progress.address].CountSeen = progress.uniqueInvSeen
		case address := <-l.closeC:
			l.status[address].Active = false
			log.Printf("Lost connection with %s", address)
		}
	}
}

func (l *Listener) printProgress() {
	sum := 0
	nodes := 0
	for _, m := range l.status {
		sum += m.CountSeen
		if m.CountSeen != 0 {
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

// NewListener creates a new listener from an existing crawler image and run the listener for a fixed duration of time.
func NewListener(image *Image, duration time.Duration) *Listener {
	l := new(Listener)
	l.progressC = make(chan *watchProgress)
	l.closeC = make(chan string)
	l.DoneC = make(chan struct{})
	l.status = make(map[string]*nodeStatus)
	l.addrC = make(chan []*wire.NetAddress)

	l.onlineNodes = image.OnlineNodes()
	l.duration = duration
	return l
}
