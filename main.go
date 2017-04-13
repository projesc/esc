package main

var config *Config

func main() {
	config = LoadConfig()

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
