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
	Payload []byte
}

func handler(_ string, request interface{}) interface{} {
	message := request.(*Message)
	log.Printf("Received %s from %s\n", message.Name, message.From)
	return request
}

func sendEvent(config *Config, to string, name string, payload []byte) {
	log.Printf("Sending event %s to %s\n", name, to)
	send(config, &Message{
		To:      to,
		Name:    name,
		Event:   true,
		Payload: payload,
	})
}

func send(config *Config, msg *Message) {
	msg.From = config.Self

	if msg.To == "*" {
		for name, node := range config.Nodes {
			msg.To = name
			_, cerr := node.Client.Call(msg)
			if cerr != nil {
				log.Println(cerr)
			}
		}
	} else {
		_, cerr := config.Nodes[msg.To].Client.Call(msg)
		if cerr != nil {
			log.Println(cerr)
		}
	}
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
			c.Start()
			node.Client = c
			sendEvent(config, node.Service.Name, "ping", []byte("ping"))
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
