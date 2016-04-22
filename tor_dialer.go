package main

import (
	"encoding/base32"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"strings"
)

const (
	PROXY_ADDR = "127.0.0.1:9050"
)

// DialTor will dial an IP address through Tor.
func DialTor(network string, tcpAddr *net.TCPAddr) (net.Conn, error) {
	dialer, err := proxy.SOCKS5("tcp", PROXY_ADDR, nil, proxy.Direct)

	if err != nil {
		log.Println("Tor seems to be down...", err)
		return nil, err
	}

	conn, err := dialer.Dial("tcp", IpToOnion(tcpAddr))

	if err != nil {
		// log.Println("Tor dial error: ", err)
		return nil, err
	}

	return conn, nil
}

// TorUp asserts Tor is running.
func TorUp() bool {
	_, err := proxy.SOCKS5("tcp", PROXY_ADDR, nil, proxy.Direct)

	if err != nil {
		log.Println("Tor seems to be down...", err)
		return false
	}

	log.Println("Connected to Tor...")
	return true
}

// OnionToIP converts an onion string address to an IP address struct
func OnionToIp(address string) (net.IP, error) {
	cleanedAddr := strings.ToUpper(strings.Split(address, ".onion")[0])
	tail, err := base32.StdEncoding.DecodeString(cleanedAddr)
	if err != nil {
		return nil, err
	}
	tail = append(net.ParseIP("FD87:d87e:eb43::")[:6], tail...)
	return tail, nil
}

// IpToOnion converts an IP address struct to an onion address string.
func IpToOnion(tcpAddr *net.TCPAddr) string {
	onionData := strings.ToLower(base32.StdEncoding.EncodeToString(tcpAddr.IP[6:]))
	return onionData + ".onion:" + fmt.Sprintf("%d", tcpAddr.Port)
}
