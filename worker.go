package main

func processNode(n *Node) error {
	defer n.Close()

	err := n.Connect()
	if err != nil {
		return err
	}

	err = n.Handshake()
	if err != nil {
		return err
	}

	err = n.GetAddr()
	if err != nil {
		return err
	}

	n.Online = true
	return nil
}

func searcher(jobs <-chan *Node, results chan<- *Node) {
	for {
		n, ok := <-jobs

		if !ok {
			close(results)
			return
		}

		err := processNode(n)

		if err != nil {
			// log.Print("Processing error failure", err)
		}

		results <- n
	}
}
