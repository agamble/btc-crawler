package main

import (
	"fmt"
	"testing"
)

func TestImageMarshalJSON(t *testing.T) {
	n1 := NewNodeFromString("192.168.1.1:8333")
	n2 := NewNodeFromString("192.168.1.1:8333")
	n3 := NewNodeFromString("192.168.1.1:8333")

	image := NewImage()

	image.AddOnlineNode(n1)
	image.AddOfflineNode(n2)
	image.AddOfflineNode(n3)

	json, err := image.MarshalJSON()

	if err != nil {
		panic(err)
	}

	fmt.Println(string(json))

}
