package main

import (
	"fmt"
	"github.com/diogok/gorpc"
)

func main() {
	config := LoadConfig()

	nodeIn := make(chan *Node)
	nodeOut := make(chan string)

	if config.Discovery != 0 {
		startDiscovery(config, nodeIn, nodeOut)
	}

	startMessaging(config, nodeIn, nodeOut)
	registerBuiltin(config, nodeIn, nodeOut)

	if config.Join != "" {
		join(config, nodeIn, nodeOut)
	}

	end := make(chan bool, 1)
	<-end
}

func join(config *Config, nodeIn chan *Node, nodeOut chan string) {
	msg := Message{
		To:      "*",
		From:    Self(config),
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
