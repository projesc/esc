package esc

import (
	"fmt"
	"github.com/diogok/gorpc"
	"github.com/patrickmn/go-cache"
	"log"
	"time"
)

var sendQueue chan *Message
var handleQueue chan *Message
var listenerQueue chan *Listener

var listeners []*Listener

func Off(listener *Listener) {
	for i, registered := range listeners {
		if registered == listener {
			listeners = append(listeners[:i], listeners[i+1:]...)
		}
	}
}

func On(from string, name string, handler func(message *Message)) *Listener {
	listener := Listener{From: from, Name: name, Handler: handler}
	listenerQueue <- &listener
	return &listener
}

func handle(message *Message) {
	log.Printf("Received %s from %s\n", message.Name, message.From)

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

func Send(to string, name string, payload string) {
	SendC(to, name, payload, true)
}

func SendC(to string, name string, payload string, coalesce bool) {
	log.Printf("Queue %s to %s\n", name, to)

	msg := Message{
		To:       to,
		Name:     name,
		Payload:  payload,
		Coalesce: coalesce,
	}

	sendQueue <- &msg
}

func should(recent *cache.Cache, msg *Message) (should bool) {
	should = true
	if msg.Coalesce {
		full := fmt.Sprintf("%s,%s,%s", msg.To, msg.Name, msg.Payload)
		_, shouldNot := recent.Get(full)
		recent.Set(full, msg.Name, cache.DefaultExpiration)
		if shouldNot {
			should = false
		}
	}
	return should
}

func startMessaging(nodeIn <-chan *Service) {
	sendQueue = make(chan *Message, 4)
	handleQueue = make(chan *Message, 4)
	listenerQueue = make(chan *Listener, 4)

	gorpc.RegisterType(&Message{})

	s := gorpc.NewTCPServer(fmt.Sprintf("0.0.0.0:%d", Config().Port), func(_ string, req interface{}) interface{} {
		message := req.(*Message)
		handleQueue <- message
		return nil
	})

	err := s.Start()
	if err != nil {
		panic(err)
	}

	clients := make(map[string]*gorpc.Client)
	recentSend := cache.New(2*time.Second, 2*time.Second)
	recentHandle := cache.New(2*time.Second, 2*time.Second)
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for {
			select {
			case listener := <-listenerQueue:
				listeners = append(listeners, listener)
			case service := <-nodeIn:
				log.Printf("Node in %s\n", service.Name)
				c := gorpc.NewTCPClient(fmt.Sprintf("%s:%d", service.AddrV4.String(), service.Port))
				c.Start()
				clients[service.Name] = c
				go Send("*", "connected", service.Name)
			case msg := <-handleQueue:
				if should(recentHandle, msg) {
					log.Println("Handling", msg.Name, msg.From)
					go handle(msg)
				} else {
					log.Println("Not handling", msg.Name, msg.From)
				}
			case msg := <-sendQueue:
				msg.From = Self()
				if should(recentSend, msg) {
					log.Println("Sending", msg.Name, msg.To)
					if msg.To == "*" {
						for _, c := range clients {
							go c.Call(msg)
						}
					} else {
						if c, ok := clients[msg.To]; ok {
							go c.Call(msg)
						}
					}
				} else {
					log.Println("Not sending", msg.Name, msg.To)
				}
			case <-ticker.C:
				for name, client := range clients {
					snap := client.Stats.Snapshot()
					if snap.ReadErrors > 10 || snap.WriteErrors > 10 || snap.AcceptErrors > 10 || snap.DialErrors > 10 {
						log.Printf("Node out %s\n", name)
						client.Stop()
						delete(clients, name)
					}
				}
			}
		}
	}()
}
