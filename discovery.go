package main

import (
	"fmt"
	"github.com/micro/mdns"
	"log"
	"strings"
	"time"
)

func Self(config *Config) string {
	return fmt.Sprintf("%s._dsc._tcp.local.", config.Node)
}

func NameOf(node string) string {
	return fmt.Sprintf("%s._dsc._tcp.local.", node)
}

func startDiscovery(config *Config, nodeIn chan *Node, nodeOut chan string) chan bool {
	config.Self = Self(config)
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
			if strings.HasSuffix(entry.Name, "_dsc._tcp.local.") {
				if _, ok := config.Nodes[entry.Name]; ok {
				} else {
					log.Printf("Found node %s\n", entry.Name)
					node := Node{
						Service: entry,
					}
					config.Nodes[node.Service.Name] = &node
					nodeIn <- &node
				}
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
