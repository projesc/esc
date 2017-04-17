package main

var config *Config

func main() {
	config = LoadConfig()

	nodeIn := startDiscovery()
	startMessaging(nodeIn)
	registerBuiltin()

	if config.Scripts != "" {
		startScripting()
		startDirSync()
	}

	end := make(chan bool, 1)
	<-end

}
