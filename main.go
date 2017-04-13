package main

import (
	"sync"
)

var config *Config
var configLock *sync.RWMutex

func main() {
	config = LoadConfig()
	configLock = &sync.RWMutex{}
	nodeIn := make(chan *Node)

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
