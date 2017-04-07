package main

import (
	"github.com/patrickmn/go-cache"
	"log"
	"strings"
	"time"
)

var kv *cache.Cache

func pingCmd(config *Config, message *Message) {
	SendEvent(config, "ping", "ping")
}

func pingEvt(config *Config, message *Message) {
	log.Printf("Ping from %s\n", message.From)
}

func setEvt(config *Config, msg *Message) {
	parts := strings.Split(msg.Payload, ",")
	if len(parts) == 2 {
		log.Printf("Set %s = %s", parts[0], parts[1])
		kv.Set(parts[0], parts[1], cache.NoExpiration)
	}
}

func membersCmd(config *Config, message *Mesage) {
}

func memberEvt(config *Config, message *Message) {
}

func registerBuiltin(config *Config) {
	kv = cache.New(6*time.Hour, 1*time.Hour)

	OnCommand(config, "*", "ping", pingCmd)
	OnEvent(config, "*", "ping", pingEvt)

	OnEvent(config, "*", "set", setEvt)
}
