package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

func searcher(jobs <-chan *net.TCPAddr, results chan<- *Node) {
	for {
		n := NewNode(<-jobs)

		err := n.Connect()
		if err != nil {
			n.Close()
			results <- nil
			continue
		}

		err = n.Handshake()
		if err != nil {
			n.Close()
			results <- nil
			continue
		}

		err = n.GetAddr()
		if err != nil {
			n.Close()
			results <- nil
			continue
		}

		fmt.Println("Returning successful node connection")

		results <- n
		n.Close()
	}
}

func writeNodeToJson(node *Node, file *os.File) error {
	out, err := json.Marshal(node)

	if err != nil {
		log.Println(err)
		return err
	}

	file.Write(out)
	file.WriteString("\n")
	file.Sync()

	return nil
}

func main() {
	jobs := make(chan *net.TCPAddr, 1000000)
	results := make(chan *Node, 100)
	seen := make(map[string]bool)

	file, err := os.Create("snapshot.json")

	if err != nil {
		log.Print(err)
	}

	defer file.Close()

	seed, err := net.ResolveTCPAddr("tcp", "148.251.238.178:8333")

	if err != nil {
		log.Print(err)
	}

	jobs <- seed
	seen[seed.String()] = true

	for i := 0; i < 1000; i++ {
		go searcher(jobs, results)
	}

	count := 0
	countFailed := 0
	for {
		node := <-results

		if node == nil {
			countFailed++
			continue
		}

		count++

		for _, tcpAddr := range node.Adjacents {
			if !seen[tcpAddr.String()] {
				seen[tcpAddr.String()] = true
				jobs <- tcpAddr
			}
		}

		err := writeNodeToJson(node, file)

		if err != nil {
			log.Println("Error marshaling to JSON")
		}

		fmt.Println("Jobs available:", len(jobs))
		fmt.Println("Nodes processed:", count)
		fmt.Println("Nodes failed:", countFailed)
	}
}
