package main

import (
	"encoding/gob"
	"encoding/json"
	"github.com/btcsuite/btcd/wire"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

type Decoder struct {
	dir string
}

type sightingHolder struct {
	Address  string
	Sighting *StampedSighting
}

type timeSlice []*sightingHolder

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].Sighting.Timestamp.Before(p[j].Sighting.Timestamp)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (d *Decoder) processFile(name string) []*StampedSighting {
	f, err := os.Open(path.Join(d.dir, name))
	defer f.Close()

	if err != nil {
		panic(err)
	}

	dec := gob.NewDecoder(f)
	sightings := make([]*StampedSighting, 0)

	for {
		shellSighting := StampedSighting{}
		err = dec.Decode(&shellSighting)

		if err != nil {
			return sightings
		}

		sightings = append(sightings, &shellSighting)
	}

	return nil
}

// Convert the binary encoded data from the listener output into a human readable JSON format
func (d *Decoder) Decode() {
	gob.Register(StampedSighting{})
	gob.Register(wire.InvVect{})

	files, err := ioutil.ReadDir(d.dir)
	if err != nil {
		panic(err)
	}

	sightings := make(timeSlice, 0)

	for _, file := range files {
		if strings.Contains(file.Name(), "block") {
			stampedSightings := d.processFile(file.Name())
			for _, s := range stampedSightings {
				address := strings.Split(file.Name(), "-")
				holder := sightingHolder{Address: address[0], Sighting: s}
				sightings = append(sightings, &holder)
			}
		}
	}

	sort.Sort(sightings)

	f, err := os.Create("timeline.json")
	enc := json.NewEncoder(f)
	enc.Encode(sightings)

}

func NewDecoder(dir string) *Decoder {
	dec := new(Decoder)
	dec.dir = dir
	return dec
}
