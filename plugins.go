package esc

import (
	"log"
	"plugin"
	"strings"
)

var plugins map[string]*Plugin

func startPlugins() {
	plugins = make(map[string]*Plugin)
	detected := make(chan string, 4)
	start := make(chan string, 4)
	stop := make(chan string, 4)

	OnEvent(Self(), "fileSync", func(msg *Message) {
		parts := strings.SplitN(msg.Payload, ",", 2)
		file := parts[0]
		if strings.HasSuffix(file, ".so") {
			detected <- file
		}
	})

	OnEvent(Self(), "fileRemoved", func(msg *Message) {
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
				SendEvent("pluginStopped", plugin.File)
			case file := <-start:
				log.Println("Starting plugin", file)

				plug, err := plugin.Open(file)
				if err != nil {
					log.Println("Open plugin error:", err)
					continue
				}

				start, serr0 := plug.Lookup("Start")
				if serr0 != nil {
					log.Println("Lookup start error:", serr0)
					continue
				}

				stop, serr2 := plug.Lookup("Stop")
				if serr2 != nil {
					log.Println("Lookup stop error:", serr2)
					continue
				}

				script, serr1 := plug.Lookup("Script")
				if serr1 != nil {
					log.Println("Lookup script error:", serr1)
					continue
				}

				p := Plugin{
					Id:     RandId(),
					File:   file,
					Stop:   stop.(func()),
					Script: script.(func(*Script)),
				}

				plugins[p.Id] = &p
				start.(func(*EscConfig))(config)
				SendEvent("pluginStarted", p.File)
			}
		}
	}()
}
