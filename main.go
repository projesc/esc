package main

import (
	"github.com/micro/mdns"
)

var config *Config

func main() {
	config = LoadConfig()

	nodeIn := make(chan *mdns.ServiceEntry)

	if config.Discovery != 0 {
		startDiscovery(nodeIn)
	}

	startMessaging(nodeIn)
	registerBuiltin()

	if config.Scripts != "" {
		startScripting()
	}

	startDirSync()

	end := make(chan bool, 1)
	<-end

}
