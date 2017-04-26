package main

import (
	"crypto/rand"
	"encoding/base64"
)

var config *Config

func main() {
	config = LoadConfig()

	nodeIn := startDiscovery()
	startMessaging(nodeIn)
	registerBuiltin()

	if config.Scripts != "" {
		startScripting()
		startDirSync()
	}

	end := make(chan bool, 1)
	<-end

}

func RandId() string {
	n := 8
	b := make([]byte, n)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
