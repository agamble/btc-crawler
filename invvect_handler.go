package main

import (
	"github.com/btcsuite/btcd/wire"
	"log"
	"time"
)

type invSighting struct {
	TimeStamp time.Time
}

func invVectHandler(listener chan []*wire.InvVect) {

	seen := make(map[wire.ShaHash][]*invSighting)

	for {
		invVect := <-listener

		for _, invData := range invVect {
			invSighting := invSighting{TimeStamp: time.Now()}

			seen[invData.Hash] = append(seen[invData.Hash], &invSighting)

			log.Println("hash", invData.Hash.String())
			for _, invSighting := range seen[invData.Hash] {
				log.Println("Sighted at", invSighting.TimeStamp.String())
			}
		}

	}
}
