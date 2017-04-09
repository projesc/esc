package main

import (
	"fmt"
	"github.com/micro/mdns"
	"github.com/patrickmn/go-cache"
	"log"
	"net"
	"strings"
	"time"
)

var kv *cache.Cache

func Get(key string) string {
	v, found := kv.Get(key)
	if !found {
		log.Println(key, "not found")
		return ""
	}
	return v.(string)
}

func Set(key string, value string) {
	kv.Set(key, value, cache.NoExpiration)
	SendEvent("set", fmt.Sprintf("%s,%s", key, value))
}

func pingCmd(message *Message) {
	SendEvent("ping", "ping")
}

func pingEvt(message *Message) {
	log.Printf("Ping from %s\n", message.From)
}

func setEvt(msg *Message) {
	if msg.From != Self() {
		parts := strings.SplitN(msg.Payload, ",", 2)
		if len(parts) == 2 {
			log.Printf("Set %s = %s", parts[0], parts[1])
			kv.Set(parts[0], parts[1], cache.NoExpiration)
		}
	}
}

func joinCmd(message *Message, nodeIn chan *Node) {
	ip := net.ParseIP(message.Payload)
	service := mdns.ServiceEntry{
		Name:   message.From,
		AddrV4: ip,
	}
	node := Node{
		Service: &service,
	}
	config.Nodes[message.From] = &node
	nodeIn <- &node
}

func registerBuiltin(nodeIn chan *Node, _ chan string) {
	kv = cache.New(6*time.Hour, 1*time.Hour)

	OnCommand("*", "ping", pingCmd)
	OnEvent("*", "ping", pingEvt)
	OnEvent("*", "set", setEvt)
	OnCommand("*", "join", func(message *Message) {
		joinCmd(message, nodeIn)
	})

}
