package main

import ()

func main() {
	config := LoadConfig()

	serviceIn := make(chan string)

	startDiscovery(config, serviceIn)
	startMessaging(config, serviceIn)

	end := make(chan bool, 1)
	<-end
}
