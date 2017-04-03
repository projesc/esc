package main

import (
	"fmt"
	"github.com/valyala/gorpc"
	"log"
)

type Message struct {
	From    string
	To      string
	Name    string
	Payload string
}

func startMessaging(config *Config, servers chan string) chan bool {
	quit := make(chan bool)

	gorpc.RegisterType(&Message{})

	d := gorpc.NewDispacher()

	d.AddFunc("Command", func(message Message) {
		log.Println(message.Name)
	})

	d.AddFunc("Event", func(message Message) {
		log.Println(message.Name)
	})

	s := gorpc.NewTCPServer(fmt.Sprintf("0.0.0.0:%d", config.Port), d.NewHandlerFunc())
	err := s.Start()

	if err != nil {
		panic(err)
	}

	var serverMap map[string]*gorpc.Client

	go func() {
		for name := range servers {
			c := gorpc.NewTCPClient("127.0.0.1:12445")
			c.Start()
			serverMap[name] = c
		}
	}()

	go func() {
		<-quit
		s.Stop()
		for _, server := range serverMap {
			server.Stop()
		}
	}()

	return quit
}
