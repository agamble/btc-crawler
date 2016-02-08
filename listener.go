package main

import (
	"github.com/btcsuite/btcd/wire"
	"time"
)

func receiveMessage(node *Node, listener chan *wire.MsgInv) {
	err := node.Connect()
	if err != nil {
		return
	}

	err = node.Handshake()
	if err != nil {
		return
	}

	msg, err := node.Inv()
	if err != nil {
		return
	}

	listener <- msg
}

func listener(node *Node, invVectHandler chan []*wire.InvVect) {

	ticker := time.NewTicker(time.Second * 10)
	result := make(chan *wire.MsgInv, 1)
	for {
		select {
		case <-ticker.C:
			go receiveMessage(node, result)
		case message := <-result:
			invVect := message.InvList
			invVectHandler <- invVect
		}
	}

}
