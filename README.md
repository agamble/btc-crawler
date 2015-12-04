# btc-crawler
A Bitcoin network crawler in Go.

To run:

    go get github.com/agamble/btc-crawler
    
    btc-crawler
    
Make sure your $GOPATH and $PATH are set correctly ($PATH should include your $GOBIN folder).

You may want to redirect output to a file and stderr to /dev/null - net.Conn spits out a lot of errors if it can't connect or a peer sends EOF.

You may need to adjust ulimit -n so that you can operate on 1000+ connections simultaneously.

With around 4000 workers I get a full network traverse in around 10-15 minutes (~4800 connectable, ~300k non-connectable/invalid) using around 150MB of memory. I did this test on a 512MB DigitalOcean box.
