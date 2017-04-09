package main

import (
	"fmt"
	"github.com/diogok/gorpc"
)

var config *Config

func main() {
	config = LoadConfig()

	nodeIn := make(chan *Node)
	nodeOut := make(chan string)

	if config.Discovery != 0 {
		startDiscovery(nodeIn, nodeOut)
	}

	startMessaging(nodeIn, nodeOut)
	registerBuiltin(nodeIn, nodeOut)

	if config.Scripts != "" {
		startScripting()
	}

	startDirSync(nodeIn)

	if config.Join != "" {
		join(nodeIn, nodeOut)
	}

	end := make(chan bool, 1)
	<-end
}

func join(nodeIn chan *Node, nodeOut chan string) {
	msg := Message{
		To:      "*",
		From:    Self(),
		Command: true,
		Event:   false,
		Name:    "join",
		Payload: config.IPs[0].String(),
	}
	c := gorpc.NewTCPClient(fmt.Sprintf("%s:%d", config.Join, config.Port))
	c.Start()
	c.Call(&msg)
	c.Stop()
}
