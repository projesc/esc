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

		if listener.From != "" && listener.From != "*" && listener.From != message.From {
			ok = false
		}
		if listener.Name != "" && listener.Name != "*" && listener.Name != message.Name {
			ok = false
		}

		if ok {
			listener.Handler(message)
		}
	}
}

func SendCommand(to string, name string, payload []byte) {
	SendCommandC(to, name, payload, true)
}

func SendCommandC(to string, name string, payload []byte, coalesce bool) {
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

func SendEvent(name string, payload []byte) {
	SendEventC(name, payload, false)
}

func SendEventC(name string, payload []byte, coalesce bool) {
	log.Printf("Sending event %s\n", name)

	msg := Message{
		To:       "*",
		Name:     name,
		Event:    true,
		Command:  false,
		Payload:  payload,
		Coalesce: coalesce,
	}

	sendQueue <- &msg
}

func sendSingle(to string, msg *Message) {
	if node, gotNode := config.Nodes[to]; gotNode {
		if node.Client != nil {
			_, cerr := node.Client.Call(msg)
			if cerr != nil {
				log.Println(cerr)
			}
			log.Println("Sent", msg.Name)
		} else {
			log.Println("No client", msg.To)
		}
	} else {
		log.Println("No node", msg.To)
	}
}

func send(msg *Message) {
	configLock.RLock()
	msg.From = Self()
	if msg.To == "*" {
		for name, _ := range config.Nodes {
			sendSingle(name, msg)
		}
	} else {
		sendSingle(msg.To, msg)
	}
	configLock.RUnlock()
}

func startMessaging(nodeIn <-chan *Node) {
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
				full := fmt.Sprintf("%s,%s", msg.Name, msg.Payload)
				_, shouldNotHandle := recentHandle.Get(full)
				recentHandle.Set(full, msg.Name, cache.DefaultExpiration)
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
				full := fmt.Sprintf("%s,%s,%s", msg.To, msg.Name, msg.Payload)
				_, shouldNotSend := recentSend.Get(full)
				recentSend.Set(full, msg.Name, cache.DefaultExpiration)
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
			configLock.Lock()
			config.Nodes[node.Service.Name] = node
			configLock.Unlock()
			SendCommand(node.Service.Name, "ping", []byte("ping"))
			SendEvent("connected", []byte(node.Service.Name))
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C
			configLock.RLock()
			for name, node := range config.Nodes {
				snap := node.Client.Stats.Snapshot()
				if snap.ReadErrors > 10 || snap.WriteErrors > 10 || snap.AcceptErrors > 10 || snap.DialErrors > 10 {
					log.Printf("Node out %s\n", name)
					config.Nodes[name].Client.Stop()
					delete(config.Nodes, name)
				}
			}
			configLock.RUnlock()
		}
	}()
}
