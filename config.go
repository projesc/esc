package esc

import (
	"flag"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"log"
	"net"
	"os"
)

var config *EscConfig

func defaultConfig() EscConfig {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	config := EscConfig{
		Node:      host,
		Host:      host,
		Port:      8901,
		Discovery: 8902,
		IFace:     "eth0",
		Directory: "files",
		Extras:    make(map[string]string),
	}

	return config
}

func Config() *EscConfig {
	if config != nil {
		return config
	}
	config = &EscConfig{}

	defaultConfig := defaultConfig()

	configFile := "config.yml"
	if len(os.Args) > 1 && len(os.Args)%2 == 0 {
		configFile = os.Args[len(os.Args)-1]
	}

	_, err := os.Stat(configFile)
	var fileConfig EscConfig
	if err == nil {
		content, err0 := ioutil.ReadFile(configFile)
		if err0 != nil {
			panic(err0)
		}

		err1 := yaml.Unmarshal(content, &fileConfig)
		if err1 != nil {
			panic(err1)
		}

		if fileConfig.Node != "" {
			defaultConfig.Node = fileConfig.Node
		}
		if fileConfig.Join != "" {
			defaultConfig.Join = fileConfig.Join
		}
		if fileConfig.IFace != "" {
			defaultConfig.IFace = fileConfig.IFace
		}
		if fileConfig.Discovery != 0 {
			defaultConfig.Discovery = fileConfig.Discovery
		}
		if fileConfig.Port != 0 {
			defaultConfig.Port = fileConfig.Port
		}
		if fileConfig.Directory != "" {
			defaultConfig.Directory = fileConfig.Directory
		}
		if fileConfig.Extras != nil {
			config.Extras = fileConfig.Extras
		}
	} else {
		log.Println(err)
	}

	flag.StringVar(&config.Node, "node", defaultConfig.Node, "Name of this node")
	flag.StringVar(&config.Join, "join", defaultConfig.Join, "Address of node to join")
	flag.StringVar(&config.IFace, "iface", defaultConfig.IFace, "Network Interface to bind to")
	flag.StringVar(&config.Directory, "dir", defaultConfig.Directory, "Plugin/Directory directory")
	flag.IntVar(&config.Discovery, "discovery", defaultConfig.Discovery, "Port for network discovery")
	flag.IntVar(&config.Port, "port", defaultConfig.Port, "Port for cluster conns")
	flag.Parse()

	iface, errFace := net.InterfaceByName(config.IFace)
	if errFace != nil {
		panic(errFace)
	}
	config.Net = iface

	addrs, aerr := config.Net.Addrs()
	if aerr != nil {
		panic(aerr)
	}

	for _, addr := range addrs {
		ip, _, iperr := net.ParseCIDR(addr.String())
		if iperr != nil {
			log.Println(iperr)
		} else {
			log.Println(ip)
			config.IPs = append(config.IPs, ip)
		}
	}

	log.Printf("%v\n", config)
	return config
}
