package main

import (
	"github.com/yuin/gopher-lua"
	"log"
	"strings"
	"time"
)

var calls chan *Call
var callbacks chan *Callback

type Script struct {
	Lua       *lua.LState
	Listeners []*Listener

	Active bool

	File string
	Hash string
}

type Call struct {
	Message *Message
	Handler *lua.LFunction
	Script  *Script
}

type Callback struct {
	Back   chan bool
	Script *Script
	Fun    *lua.LFunction
}

func stopScript(script *Script) {
	script.Active = false
	for _, listener := range script.Listeners {
		Off(listener)
	}
	script.Lua.Close()
}

func luaMessage(vm *lua.LState, message *Message) *lua.LTable {
	table := vm.NewTable()
	vm.SetField(table, "from", lua.LString(message.From))
	vm.SetField(table, "name", lua.LString(message.Name))
	vm.SetField(table, "payload", lua.LString(string(message.Payload)))
	return table
}

func luaSendCmd(vm *lua.LState) int {
	to := vm.ToString(1)
	name := vm.ToString(2)
	payload := vm.ToString(3)
	SendCommand(to, name, []byte(payload))
	return 0
}

func luaSendCmdC(vm *lua.LState) int {
	to := vm.ToString(1)
	name := vm.ToString(2)
	payload := vm.ToString(3)
	coalesce := vm.ToBool(4)
	SendCommandC(to, name, []byte(payload), coalesce)
	return 0
}

func luaSendEvt(vm *lua.LState) int {
	name := vm.ToString(1)
	payload := vm.ToString(2)
	SendEvent(name, []byte(payload))
	return 0
}

func luaOnCmd(script *Script, vm *lua.LState) int {
	from := vm.ToString(1)
	name := vm.ToString(2)
	handler := vm.ToFunction(3)

	listener := OnCommand(from, name, func(message *Message) {
		calls <- &Call{message, handler, script}
	})

	script.Listeners = append(script.Listeners, listener)
	return 0
}

func luaOnEvt(script *Script, vm *lua.LState) int {
	from := vm.ToString(1)
	name := vm.ToString(2)
	handler := vm.ToFunction(3)

	listener := OnEvent(from, name, func(message *Message) {
		calls <- &Call{message, handler, script}
	})

	script.Listeners = append(script.Listeners, listener)
	return 0
}

func luaLog(vm *lua.LState) int {
	text := vm.ToString(1)
	log.Println(text)
	return 0
}

func luaGet(vm *lua.LState) int {
	key := vm.ToString(1)
	value := Get(key)
	vm.Push(lua.LString(value))
	return 1
}

func luaSet(vm *lua.LState) int {
	key := vm.ToString(1)
	value := vm.ToString(2)
	Set(key, value)
	return 0
}

func luaSelf(vm *lua.LState) int {
	vm.Push(lua.LString(Self()))
	return 1
}

func luaNameOf(vm *lua.LState) int {
	vm.Push(lua.LString(NameOf(vm.ToString(1))))
	return 1
}

func luaConfig(vm *lua.LState) int {
	vm.Push(lua.LString(config.Extras[vm.ToString(1)]))
	return 1
}

func luaTick(script *Script, vm *lua.LState) int {
	sec := vm.ToInt(1)
	fun := vm.ToFunction(2)

	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	go func() {
		for {
			back := make(chan bool, 1)
			callback := Callback{Script: script, Fun: fun, Back: back}
			callbacks <- &callback
			ok := <-back
			if ok && script.Active {
				<-ticker.C
			} else {
				return
			}
		}
	}()
	return 0
}

func startScript(file string) *Script {
	vm := lua.NewState()

	script := Script{
		Lua:    vm,
		File:   file,
		Active: true,
	}

	vm.SetGlobal("nameOf", vm.NewFunction(luaNameOf))
	vm.SetGlobal("config", vm.NewFunction(luaConfig))
	vm.SetGlobal("self", vm.NewFunction(luaSelf))
	vm.SetGlobal("log", vm.NewFunction(luaLog))
	vm.SetGlobal("set", vm.NewFunction(luaSet))
	vm.SetGlobal("get", vm.NewFunction(luaGet))
	vm.SetGlobal("onEvent", vm.NewFunction(func(vm *lua.LState) int {
		return luaOnEvt(&script, vm)
	}))
	vm.SetGlobal("onCommand", vm.NewFunction(func(vm *lua.LState) int {
		return luaOnCmd(&script, vm)
	}))
	vm.SetGlobal("tick", vm.NewFunction(func(vm *lua.LState) int {
		return luaTick(&script, vm)
	}))
	vm.SetGlobal("sendEvent", vm.NewFunction(luaSendEvt))
	vm.SetGlobal("sendCommand", vm.NewFunction(luaSendCmd))
	vm.SetGlobal("sendCommandC", vm.NewFunction(luaSendCmdC))

	err := vm.DoFile(file)
	if err != nil {
		log.Println(err)
		stopScript(&script)
	}

	return &script
}

func startScripting() {
	scripts := make(map[string]*Script)

	detected := make(chan string, 4)
	start := make(chan string, 4)
	stop := make(chan string, 4)

	calls = make(chan *Call, 4)
	callbacks = make(chan *Callback, 4)

	OnEvent(Self(), "fileSync", func(msg *Message) {
		parts := strings.SplitN(string(msg.Payload), ",", 2)
		detected <- parts[0]
	})

	OnEvent(Self(), "fileRemoved", func(msg *Message) {
		name := string(msg.Payload)
		if _, ok := scripts[name]; ok {
			stop <- name
		}
	})

	go func() {
		for {
			select {
			case name := <-detected:
				if strings.HasSuffix(name, ".lua") {
					if _, ok := scripts[name]; ok {
						stop <- name
					}
					start <- name
				}
			case name := <-stop:
				log.Println("Stoping script", name)
				stopScript(scripts[name])
				delete(scripts, name)
			case name := <-start:
				log.Println("Starting script", name)
				scripts[name] = startScript(name)
			case call := <-calls:
				err := call.Script.Lua.CallByParam(lua.P{
					Fn:      call.Handler,
					NRet:    0,
					Protect: true,
				}, luaMessage(call.Script.Lua, call.Message))
				if err != nil {
					log.Println(err)
				}
			case call := <-callbacks:
				err := call.Script.Lua.CallByParam(lua.P{
					Fn:      call.Fun,
					NRet:    1,
					Protect: true,
				})
				if err != nil {
					log.Println(err)
					call.Back <- false
				} else {
					call.Back <- call.Script.Lua.Get(-1) == lua.LTrue
				}
			}
		}
	}()
}
