package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/yuin/gopher-lua"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Script struct {
	Lua       *lua.LState
	Listeners []*Listener

	File string
	Hash string
}

func stopScript(script *Script) {
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
	coalesce := vm.ToBool(4)
	SendCommand(to, name, payload, coalesce)
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
		err := vm.CallByParam(lua.P{
			Fn:      handler,
			NRet:    0,
			Protect: true,
		}, luaMessage(vm, message))
		if err != nil {
			log.Println(err)
		}
	})

	script.Listeners = append(script.Listeners, listener)

	return 0
}

func luaOnEvt(script *Script, vm *lua.LState) int {
	from := vm.ToString(1)
	name := vm.ToString(2)
	handler := vm.ToFunction(3)

	listener := OnEvent(from, name, func(message *Message) {
		err := vm.CallByParam(lua.P{
			Fn:      handler,
			NRet:    0,
			Protect: true,
		}, luaMessage(vm, message))
		if err != nil {
			log.Println(err)
		}
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

func luaTicker(vm *lua.LState) int {
	sec := vm.ToInt(1)
	fun := vm.ToFunction(2)
	go func() {
		for {
			err := vm.CallByParam(lua.P{
				Fn:      fun,
				NRet:    1,
				Protect: true,
			})
			if err != nil {
				log.Println(err)
				return
			} else {
				ret := vm.Get(-1)
				if ret == lua.LTrue {
					time.Sleep(time.Duration(sec) * time.Second)
				} else {
					return
				}
			}
		}
	}()
	return 0
}

func startScript(file string) *Script {
	vm := lua.NewState()

	script := Script{
		Lua:  vm,
		File: file,
	}

	vm.SetGlobal("ticker", vm.NewFunction(luaTicker))
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
	vm.SetGlobal("sendEvent", vm.NewFunction(luaSendEvt))
	vm.SetGlobal("sendCommand", vm.NewFunction(luaSendCmd))

	err := vm.DoFile(fmt.Sprintf("%s/%s", config.Scripts, file))
	if err != nil {
		log.Println(err)
		vm.Close()
	}

	return &script
}

func startScripting() {
	if config.Scripts == "" {
		return
	}

	vms := make(map[string]*Script)

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			<-ticker.C
			_, errStat := os.Lstat(config.Scripts)
			if errStat != nil {
				log.Println(errStat)
				continue
			}
			dir, _ := os.Open(config.Scripts)

			files, err := dir.Readdir(0)

			if err != nil {
				log.Println(err)
				continue
			}

			got := make(map[string]bool)

			for _, fileInfo := range files {
				fileName := fileInfo.Name()
				if strings.HasSuffix(fileName, ".lua") {
					got[fileName] = true

					hasher := sha256.New()
					f, _ := os.Open(fmt.Sprintf("%s/%s", config.Scripts, fileName))
					io.Copy(hasher, f)
					f.Close()

					hash := hex.EncodeToString(hasher.Sum(nil))

					if _, ok := vms[fileName]; ok {
						if vms[fileName].Hash == hash {
						} else {
							stopScript(vms[fileName])
							log.Println("Restarting script", fileName)
							vms[fileName] = startScript(fileName)
							vms[fileName].Hash = hash
						}
					} else {
						log.Println("Starting script", fileName)
						vms[fileName] = startScript(fileName)
						vms[fileName].Hash = hash
					}
				}
			}

			var toRemove []string
			for name, vm := range vms {
				if _, ok := got[name]; ok {
				} else {
					log.Println("Stoping script", name)
					stopScript(vm)
					toRemove = append(toRemove, name)
				}
			}

			for _, name := range toRemove {
				delete(vms, name)
			}
		}
	}()

}
