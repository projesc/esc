package esc

import (
	"fmt"
	"github.com/micro/mdns"
	"github.com/patrickmn/go-cache"
	"log"
	"strings"
	"time"
)

func startDiscovery() chan *Service {
	nodeIn := make(chan *Service)
	service, err0 := mdns.NewMDNSService(Config().Node, ServiceName(), "", Config().Host, config.Discovery, config.IPs, []string{"esc"})
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
			if strings.HasSuffix(entry.Name, fmt.Sprintf("%s.local.", ServiceName())) {
				if _, ok := found.Get(entry.Name); !ok {
					log.Printf("Found node %s\n", entry.Name)
					nodeIn <- &Service{
						Name:   entry.Name,
						Port:   Config().Port,
						AddrV4: &entry.AddrV4,
					}
				}
				found.Set(entry.Name, entry.Name, cache.DefaultExpiration)
			}
		}
	}()

	ticker := time.NewTicker(8 * time.Second)
	go func() {
		for {
			mdns.Lookup(ServiceName(), entries)
			<-ticker.C
		}
	}()
	return nodeIn
}
