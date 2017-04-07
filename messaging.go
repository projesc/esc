package main

import (
	"fmt"
	"github.com/diogok/gorpc"
	"github.com/patrickmn/go-cache"
	"log"
	"time"
)

type Message struct {
	Command  bool
	Event    bool
	From     string
	Name     string
	To       string
	Payload  []byte
	Coalesce bool
}

var sendQueue chan *Message
var handleQueue chan *Message

func handle(config *Config, message *Message) {
	log.Printf("Received %s from %s\n", message.Name, message.From)
}

func SendCommand(config *Config, to string, name string, payload []byte, coalesce bool) {
	log.Printf("Sending command %s to %s\n", name, to)

	msg := Message{
		To:       to,
		Name:     name,
		Event:    false,
		Command:  true,
		Payload:  payload,
		Coalesce: coalesce,
	}

	sendQueue <- &msg
}

func SendEvent(config *Config, name string, payload []byte) {
	log.Printf("Sending event %s\n", name)
	msg := Message{
		To:       "*",
		Name:     name,
		Event:    true,
		Command:  false,
		Payload:  payload,
		Coalesce: false,
	}

	sendQueue <- &msg
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

	sendQueue = make(chan *Message)
	handleQueue = make(chan *Message)

	gorpc.RegisterType(&Message{})

	s := gorpc.NewTCPServer(fmt.Sprintf("0.0.0.0:%d", config.Port), func(_ string, req interface{}) interface{} {
		message := req.(*Message)
		handleQueue <- message
		return req
	})
	err := s.Start()

	if err != nil {
		panic(err)
	}

	recentSend := cache.New(2*time.Second, 2*time.Second)
	recentHandle := cache.New(2*time.Second, 2*time.Second)

	go func() {
		for msg := range handleQueue {
			shouldHandle := true
			if msg.Coalesce {
				_, shouldNotHandle := recentHandle.Get(msg.Name)
				recentHandle.Set(msg.Name, msg.Name, cache.DefaultExpiration)
				if shouldNotHandle {
					shouldHandle = false
				}
			}

			if shouldHandle {
				handle(config, msg)
			} else {
				log.Println("Dropping handle", msg.Name)
			}
		}
	}()

	go func() {
		for msg := range sendQueue {
			shouldSend := true
			if msg.Coalesce {
				_, shouldNotSend := recentSend.Get(fmt.Sprintf("%s/%s", msg.To, msg.Name))
				recentSend.Set(msg.Name, msg.Name, cache.DefaultExpiration)
				if shouldNotSend {
					shouldSend = false
				}
			}

			if shouldSend {
				send(config, msg)
			} else {
				log.Println("Dropping send", msg.Name)
			}
		}
	}()

	go func() {
		<-quit
		s.Stop()
		for _, node := range config.Nodes {
			node.Client.Stop()
		}
		close(handleQueue)
	}()

	go func() {
		for node := range nodeIn {
			c := gorpc.NewTCPClient(fmt.Sprintf("%s:%d", node.Service.AddrV4.String(), config.Port))
			c.Start()
			node.Client = c
			SendCommand(config, node.Service.Name, "ping", []byte("ping"), true)
		}
	}()

	return quit
}
