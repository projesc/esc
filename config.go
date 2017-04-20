package main

import (
	"flag"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type Config struct {
	Host      string
	Node      string `json:"node"`
	Join      string `json:"join"`
	IFace     string `json:"iface"`
	Discovery int    `json:"discovery"`
	Port      int    `json:"port"`
	Scripts   string `json:"scripts"`
	Net       *net.Interface
	IPs       []net.IP
	Extras    map[string]string `json:"extras"`
}

func defaultConfig() Config {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	config := Config{
		Node:      host,
		Host:      host,
		Port:      8901,
		Discovery: 8902,
		IFace:     "eth0",
		Scripts:   "scripts",
		Extras:    make(map[string]string),
	}

	return config
}

func LoadConfig() *Config {
	config := Config{}

	defaultConfig := defaultConfig()

	configFile := "config.yml"
	if len(os.Args) > 1 && len(os.Args)%2 == 0 {
		configFile = os.Args[len(os.Args)-1]
	}

	_, err := os.Stat(configFile)
	var fileConfig Config
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
		if fileConfig.Scripts != "" {
			defaultConfig.Scripts = fileConfig.Scripts
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
	flag.StringVar(&config.Scripts, "scripts", defaultConfig.Scripts, "Scripts directory")
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
	return &config
}
