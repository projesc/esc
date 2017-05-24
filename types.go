package esc

import (
	"github.com/yuin/gopher-lua"
	"net"
	"time"
)

type EscConfig struct {
	Host      string
	Node      string `json:"node"`
	Join      string `json:"join"`
	IFace     string `json:"iface"`
	Discovery int    `json:"discovery"`
	Port      int    `json:"port"`
	Directory string `json:"directory"`
	Net       *net.Interface
	IPs       []net.IP
	Extras    map[string]string `json:"extras"`
}

type Message struct {
	From     string `json:"from"`
	Name     string `json:"name"`
	To       string `json:"to"`
	Payload  string `json:"payload"`
	Coalesce bool   `json:"coalesce"`
}

type Service struct {
	Name   string
	AddrV4 *net.IP
	Port   int
}

type Listener struct {
	From    string
	Name    string
	Handler func(message *Message)
}

type File struct {
	Name    string
	Hash    string
	Time    time.Time
	Content []byte
}

type Script struct {
	Id        string
	Lua       *lua.LState
	Listeners []*Listener
	File      string
	Hash      string
	Done      []chan bool
}

type ScriptCall struct {
	Message *Message
	Fun     *lua.LFunction
	Script  *Script
	Done    chan bool
	Back    bool
}

type Plugin struct {
	Id     string
	File   string
	Stop   func()
	Script func(*Script)
}
