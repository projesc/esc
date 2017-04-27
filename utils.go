package esc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

func RandId() string {
	n := 8
	b := make([]byte, n)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func ServiceName() string {
	return "_esc._tcp"
}

func ShortName(fullName string) string {
	return strings.Replace(fullName, fmt.Sprintf(".%s.local", ServiceName()), "", 0)
}

func FullName(node string) string {
	return fmt.Sprintf("%s.%s.local.", node, ServiceName())
}

func Self() string {
	return fmt.Sprintf("%s.%s.local.", config.Node, ServiceName())
}

func Start() {
	Config()
	nodeIn := startDiscovery()
	startMessaging(nodeIn)
	startScripting()
	startPlugins()
	startDirSync()
}
