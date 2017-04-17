package main

import (
	"fmt"
	"github.com/micro/mdns"
	"github.com/patrickmn/go-cache"
	"log"
	"strings"
	"time"
)

var serviceName string

func Self() string {
	return fmt.Sprintf("%s.%s.local.", config.Node, serviceName)
}

func NameOf(node string) string {
	return fmt.Sprintf("%s.%s.local.", node, serviceName)
}

func startDiscovery() chan *mdns.ServiceEntry {
	nodeIn := make(chan *mdns.ServiceEntry)

	serviceName = "_esc._tcp"
	service, err0 := mdns.NewMDNSService(config.Node, serviceName, "", config.Host, config.Discovery, config.IPs, []string{"esc"})
	if err0 != nil {
		panic(err0)
	}

	_, err1 := mdns.NewServer(&mdns.Config{Zone: service, Iface: config.Net})
	if err1 != nil {
		panic(err1)
	}

	found := cache.New(24*time.Second, 16*time.Second)
	entries := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entries {
			if strings.HasSuffix(entry.Name, fmt.Sprintf("%s.local.", serviceName)) {
				if _, ok := found.Get(entry.Name); !ok {
					log.Printf("Found node %s\n", entry.Name)
					nodeIn <- entry
				}
				found.Set(entry.Name, entry.Name, cache.DefaultExpiration)
			}
		}
	}()

	ticker := time.NewTicker(8 * time.Second)
	go func() {
		for {
			mdns.Lookup(serviceName, entries)
			<-ticker.C
		}
	}()

	return nodeIn

}
