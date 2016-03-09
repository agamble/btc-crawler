package main

import "testing"

func TestTorConnect(t *testing.T) {
	seed := NewTorSeed()

	err := seed.Connect()
	if err != nil {
		panic(err)
	}
}
