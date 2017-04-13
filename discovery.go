package main

import (
	"fmt"
	"github.com/micro/mdns"
	"log"
	"strings"
	"time"
)

func Self() string {
	return fmt.Sprintf("%s._esc._tcp.local.", config.Node)
}

func NameOf(node string) string {
	return fmt.Sprintf("%s._esc._tcp.local.", node)
}

func startDiscovery(nodeIn chan<- *Node) {
	service, err0 := mdns.NewMDNSService(config.Node, "_esc._tcp", "", config.Host, config.Discovery, config.IPs, []string{"esc"})
	if err0 != nil {
		panic(err0)
	}
	_, err1 := mdns.NewServer(&mdns.Config{Zone: service, Iface: config.Net})
	if err1 != nil {
		panic(err1)
	}
	entries := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entries {
			if strings.HasSuffix(entry.Name, "_esc._tcp.local.") {
				if _, ok := config.Nodes[entry.Name]; !ok {
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
	ticker := time.NewTicker(8 * time.Second)
	go func() {
		for {
			<-ticker.C
			mdns.Lookup("_esc._tcp", entries)
		}
	}()
}
