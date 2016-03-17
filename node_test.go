package main

import (
	"testing"
)

// func TestTorConnect(t *testing.T) {
// 	seed := NewTorSeed()
//
// 	err := seed.Connect()
// 	if err != nil {
// 		panic(err)
// 	}
// }

func TestJson(t *testing.T) {
	n := NewNodeFromString("192.168.1.1:8333")
	adj := NewNodeFromString("192.168.1.1:8333")
	n.Adjacents = append(n.Adjacents, adj)

	_, err := n.MarshalJSON()

	if err != nil {
		panic(err)
	}
}
