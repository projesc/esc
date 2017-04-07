package main

func main() {
	config := LoadConfig()

	serviceIn := make(chan *Node)

	startDiscovery(config, serviceIn)
	startMessaging(config, serviceIn)
	registerBuiltin(config)

	end := make(chan bool, 1)
	<-end
}
