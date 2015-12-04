# btc-crawler
A Bitcoin network crawler

To run:

    go get github.com/agamble/btc-crawler
    
    btc-crawler
    
Make sure your $GOPATH and $PATH are set correctly ($PATH should include your $GOBIN folder)

You may want to redirect output to a file and stderr to /dev/null - net.Conn spits out a lot of errors if it can't connect or if a peer send EOF
