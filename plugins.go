package esc

import (
	"log"
	"plugin"
	"strings"
)

var plugins map[string]*Plugin
var inPlugins chan *Plugin

func startPlugins() {
	plugins = make(map[string]*Plugin)
	inPlugins = make(chan *Plugin, 4)
	detected := make(chan string, 4)
	start := make(chan string, 4)
	stop := make(chan string, 4)

	On(Self(), "fileSync", func(msg *Message) {
		parts := strings.SplitN(msg.Payload, ",", 2)
		file := parts[0]
		if strings.HasSuffix(file, ".so") {
			detected <- file
		}
	})

	On(Self(), "fileRemoved", func(msg *Message) {
		file := msg.Payload
		for _, plugin := range plugins {
			if plugin.File == file {
				stop <- plugin.Id
			}
		}
	})

	go func() {
		for {
			select {
			case file := <-detected:
				for _, plugin := range plugins {
					if plugin.File == file {
						stop <- plugin.Id
					}
				}
				start <- file
			case id := <-stop:
				plugin := plugins[id]
				log.Println("Stoping plugin", plugin.File, plugin.Id)
				plugin.Stop()
				delete(plugins, id)
				Send(Self(), "pluginStopped", plugin.File)
			case file := <-start:
				log.Println("Starting plugin", file)
				go startPlugin(file)
			case plugin := <-inPlugins:
				plugins[plugin.Id] = plugin
				Send(Self(), "pluginStarted", plugin.File)
			}
		}
	}()
}

func startPlugin(file string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in plugin", file, r)
		}
	}()

	plug, err := plugin.Open(file)
	if err != nil {
		log.Println("Open plugin error:", err)
		return
	}

	start, serr0 := plug.Lookup("Start")
	if serr0 != nil {
		log.Println("Lookup start error:", serr0)
		return
	}

	stop, serr2 := plug.Lookup("Stop")
	if serr2 != nil {
		log.Println("Lookup stop error:", serr2)
		return
	}

	script, serr1 := plug.Lookup("Script")
	if serr1 != nil {
		log.Println("Lookup script error:", serr1)
		return
	}

	p := Plugin{
		Id:     RandId(),
		File:   file,
		Stop:   stop.(func()),
		Script: script.(func(*Script)),
	}

	start.(func(*EscConfig))(config)

	inPlugins <- &p

}
