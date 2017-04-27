package esc

import (
	"github.com/cjoudrey/gluahttp"
	"github.com/yuin/gopher-lua"
	"layeh.com/gopher-json"
	"log"
	"net/http"
	"strings"
	"time"
)

var ScriptCalls chan *ScriptCall

func stopScript(script *Script) {
	for _, done := range script.Done {
		done <- true
	}
	for _, listener := range script.Listeners {
		Off(listener)
	}
	script.Lua.Close()
}

func luaMessage(vm *lua.LState, message *Message) *lua.LTable {
	table := vm.NewTable()
	vm.SetField(table, "from", lua.LString(message.From))
	vm.SetField(table, "name", lua.LString(message.Name))
	vm.SetField(table, "payload", lua.LString(message.Payload))
	return table
}

func luaSendCmd(vm *lua.LState) int {
	to := vm.ToString(1)
	name := vm.ToString(2)
	payload := vm.ToString(3)
	SendCommand(to, name, payload)
	return 0
}

func luaSendCmdC(vm *lua.LState) int {
	to := vm.ToString(1)
	name := vm.ToString(2)
	payload := vm.ToString(3)
	coalesce := vm.ToBool(4)
	SendCommandC(to, name, payload, coalesce)
	return 0
}

func luaSendEvt(vm *lua.LState) int {
	name := vm.ToString(1)
	payload := vm.ToString(2)
	SendEvent(name, payload)
	return 0
}

func luaOnCmd(script *Script, vm *lua.LState) int {
	from := vm.ToString(1)
	name := vm.ToString(2)
	handler := vm.ToFunction(3)

	listener := OnCommand(from, name, func(message *Message) {
		ScriptCalls <- &ScriptCall{Message: message, Fun: handler, Script: script}
	})

	script.Listeners = append(script.Listeners, listener)
	return 0
}

func luaOnEvt(script *Script, vm *lua.LState) int {
	from := vm.ToString(1)
	name := vm.ToString(2)
	handler := vm.ToFunction(3)

	listener := OnEvent(from, name, func(message *Message) {
		ScriptCalls <- &ScriptCall{Message: message, Fun: handler, Script: script}
	})

	script.Listeners = append(script.Listeners, listener)
	return 0
}

func luaLog(vm *lua.LState) int {
	text := vm.ToString(1)
	log.Println(text)
	return 0
}

func luaSelf(vm *lua.LState) int {
	vm.Push(lua.LString(Self()))
	return 1
}

func luaFullName(vm *lua.LState) int {
	vm.Push(lua.LString(FullName(vm.ToString(1))))
	return 1
}

func luaShortName(vm *lua.LState) int {
	vm.Push(lua.LString(ShortName(vm.ToString(1))))
	return 1
}

func luaConfig(vm *lua.LState) int {
	vm.Push(lua.LString(config.Extras[vm.ToString(1)]))
	return 1
}

func luaTick(script *Script, vm *lua.LState) int {
	sec := vm.ToInt(1)
	fun := vm.ToFunction(2)

	done := make(chan bool, 1)
	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				ScriptCalls <- &ScriptCall{Script: script, Fun: fun, Done: done, Back: true}
			case <-done:
				for i, d := range script.Done {
					if d == done {
						script.Done = append(script.Done[:i], script.Done[i+1:]...)
					}
				}
				return
			}
		}
	}()

	script.Done = append(script.Done, done)
	return 0
}

func startScript(file string) *Script {
	vm := lua.NewState()

	script := Script{
		Lua:  vm,
		File: file,
		Id:   RandId(),
	}

	vm.SetGlobal("fullName", vm.NewFunction(luaFullName))
	vm.SetGlobal("shortName", vm.NewFunction(luaShortName))
	vm.SetGlobal("config", vm.NewFunction(luaConfig))
	vm.SetGlobal("self", vm.NewFunction(luaSelf))
	vm.SetGlobal("log", vm.NewFunction(luaLog))
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

	vm.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	vm.PreloadModule("json", json.Loader)

	for _, p := range plugins {
		p.Script(&script)
	}

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

	ScriptCalls = make(chan *ScriptCall, 4)

	OnEvent(Self(), "fileSync", func(msg *Message) {
		parts := strings.SplitN(msg.Payload, ",", 2)
		file := parts[0]
		if strings.HasSuffix(file, ".lua") {
			detected <- file
		}
	})

	OnEvent(Self(), "fileRemoved", func(msg *Message) {
		name := msg.Payload
		if _, ok := scripts[name]; ok {
			stop <- name
		}
	})

	OnEvent(Self(), "pluginStarted", func(msg *Message) {
		for _, script := range scripts {
			stop <- script.Id
			start <- script.File
		}
	})

	go func() {
		for {
			select {
			case name := <-detected:
				for _, script := range scripts {
					if script.File == name {
						stop <- script.Id
					}
				}
				start <- name
			case id := <-stop:
				script := scripts[id]
				log.Println("Stoping script", script.File)
				stopScript(scripts[id])
				delete(scripts, id)
				SendEvent("scriptStopped", script.File)
			case name := <-start:
				log.Println("Starting script", name)
				script := startScript(name)
				scripts[script.Id] = script
				SendEvent("scriptStarted", script.File)
			case call := <-ScriptCalls:
				var arg *lua.LTable
				n := 0
				if call.Message != nil {
					arg = luaMessage(call.Script.Lua, call.Message)
				} else {
					n = 1
				}
				err := call.Script.Lua.CallByParam(lua.P{
					Fn:      call.Fun,
					NRet:    n,
					Protect: true,
				}, arg)
				if err != nil {
					log.Println(err)
				}
				if call.Back {
					keepOn := true
					if err != nil {
						keepOn = false
					} else {
						keepOn = call.Script.Lua.Get(-1) == lua.LTrue
					}
					if !keepOn {
						call.Done <- true
					}
				}
			}
		}
	}()
}
