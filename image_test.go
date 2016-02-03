package main

import (
	"log"
	"testing"
)

func TestDb(t *testing.T) {
	setupDb()

	seed := NewSeed()
	image := NewImage(seed)
	err := processNode(seed)

	if err != nil {
		log.Print("Processing error failure", err)
		panic(err)
	}

	for _, n := range seed.Neighbours() {
		image.Add(NewNode(n))
	}

	image.Save()
}
