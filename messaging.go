package main

import (
	"fmt"
	"github.com/valyala/gorpc"
	"log"
)

type Message struct {
	Command bool
	Event   bool
	From    string
	Name    string
	To      string
	Payload string
}

func handler(clientAddr string, request interface{}) interface{} {
	log.Printf("Obtained request %+v from the client %s\n", request, clientAddr)
	return request
}

func startMessaging(config *Config, nodeIn chan *Node) chan bool {
	quit := make(chan bool)

	gorpc.RegisterType(&Message{})

	s := gorpc.NewTCPServer(fmt.Sprintf("0.0.0.0:%d", config.Port), handler)
	err := s.Start()

	if err != nil {
		panic(err)
	}

	go func() {
		for node := range nodeIn {
			c := gorpc.NewTCPClient(fmt.Sprintf("%s:%d", node.Service.AddrV4.String(), config.Port))
			node.Client = c
			c.Start()
			_, cerr := c.Call(&Message{
				From:    config.Node,
				To:      "whatever",
				Name:    "hello",
				Payload: "hello",
			})
			if cerr != nil {
				log.Println(cerr)
			}
		}
	}()

	go func() {
		<-quit
		s.Stop()
		for _, node := range config.Nodes {
			node.Client.Stop()
		}
	}()

	return quit
}
