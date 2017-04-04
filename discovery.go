package main

import (
	"github.com/hashicorp/mdns"
	"log"
	"time"
)

func startDiscovery(config *Config, ch chan *Node) chan bool {
	info := []string{"dsc"}
	service, err0 := mdns.NewMDNSService(config.Node, "_dsc._tcp", "", config.Host, config.Discovery, config.IPs, info)
	if err0 != nil {
		panic(err0)
	}

	server, err1 := mdns.NewServer(&mdns.Config{Zone: service, Iface: config.Net})
	if err1 != nil {
		panic(err1)
	}

	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {
			if _, ok := config.Nodes[entry.Name]; ok {
			} else {
				log.Printf("Found node %s\n", entry.Name)
				server := Node{
					Name:    entry.Name,
					Service: entry,
				}
				config.Nodes[entry.Name] = server
				ch <- &server
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				mdns.Lookup("_dsc._tcp", entriesCh)
			case <-quit:
				server.Shutdown()
				return
			}
		}
	}()

	return quit
}
