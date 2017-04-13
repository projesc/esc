package main

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"log"
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
	SendEvent("set", []byte(fmt.Sprintf("%s,%s", key, value)))
}

func pingCmd(message *Message) {
	SendEvent("ping", []byte("ping"))
}

func pingEvt(message *Message) {
	log.Printf("Ping from %s\n", message.From)
}

func setEvt(msg *Message) {
	if msg.From != Self() {
		parts := strings.SplitN(string(msg.Payload), ",", 2)
		if len(parts) == 2 {
			log.Printf("Set %s = %s", parts[0], parts[1])
			kv.Set(parts[0], parts[1], cache.NoExpiration)
		}
	}
}

func registerBuiltin() {
	kv = cache.New(6*time.Hour, 1*time.Hour)

	OnCommand("*", "ping", pingCmd)
	OnEvent("*", "ping", pingEvt)
	OnEvent("*", "set", setEvt)
}
