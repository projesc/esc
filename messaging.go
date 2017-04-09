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
	Payload  string
	Coalesce bool
}

type Listener struct {
	From    string
	Name    string
	Handler func(message *Message)
}

var sendQueue chan *Message
var handleQueue chan *Message

var eventListeners []*Listener
var commandListeners []*Listener

func Off(listener *Listener) {
	for i, registered := range eventListeners {
		if registered == listener {
			eventListeners = append(eventListeners[:i], eventListeners[i+1:]...)
		}
	}
	for i, registered := range commandListeners {
		if registered == listener {
			commandListeners = append(commandListeners[:i], commandListeners[i+1:]...)
		}
	}
}

func OnEvent(from string, name string, handler func(message *Message)) *Listener {
	listener := Listener{From: from, Name: name, Handler: handler}
	eventListeners = append(eventListeners, &listener)
	return &listener
}

func OnCommand(from string, name string, handler func(message *Message)) *Listener {
	listener := Listener{From: from, Name: name, Handler: handler}
	commandListeners = append(commandListeners, &listener)
	return &listener
}

func handle(message *Message) {
	log.Printf("Received %s from %s\n", message.Name, message.From)

	var listeners []*Listener
	if message.Event {
		listeners = eventListeners
	} else if message.Command {
		listeners = commandListeners
	}

	for _, listener := range listeners {
		ok := true

		if listener.From == "" {
		} else if listener.From == "*" {
		} else if listener.From == message.From {
		} else {
			ok = false
		}

		if listener.Name == "" {
		} else if listener.Name == "*" {
		} else if listener.Name == message.Name {
		} else {
			ok = false
		}

		if ok {
			listener.Handler(message)
		}
	}
}

func SendCommand(to string, name string, payload string, coalesce bool) {
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

func SendEvent(name string, payload string) {
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

func send(msg *Message) {
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

func startMessaging(nodeIn chan *Node, nodeOut chan string) {
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
				handle(msg)
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
				send(msg)
			} else {
				log.Println("Dropping send", msg.Name)
			}
		}
	}()

	go func() {
		for node := range nodeIn {
			log.Println("New node", node.Service.Name)
			c := gorpc.NewTCPClient(fmt.Sprintf("%s:%d", node.Service.AddrV4.String(), config.Port))
			c.Start()
			node.Client = c
			SendCommand(node.Service.Name, "ping", "ping", true)
		}
	}()

	go func() {
		for nodeName := range nodeOut {
			log.Printf("Node out %s\n", nodeName)
			config.Nodes[nodeName].Client.Stop()
			delete(config.Nodes, nodeName)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C
			for name, node := range config.Nodes {
				if node.Client.Stats.Snapshot().ReadErrors > 10 {
					nodeOut <- name
				}
			}
		}
	}()
}
