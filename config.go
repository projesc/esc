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
	Host string

	Node      string `json:"node"`
	IFace     string `json:iface`
	Discovery int    `json:"discovery"`
	Port      int    `json:"port"`

	Net *net.Interface
	IPs []net.IP

	Servers map[string]net.IP
}

func defaultConfig() Config {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	config := Config{
		Node:      host,
		Host:      host,
		Port:      8181,
		Discovery: 8801,
		IFace:     "eth0",
	}
	return config
}

func LoadConfig() *Config {
	config := Config{}

	defaultConfig := defaultConfig()

	if len(os.Args) > 1 {
		configFile := os.Args[len(os.Args)-1]

		_, err := os.Stat(configFile)
		var fileConfig Config
		if !os.IsNotExist(err) {
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
			if fileConfig.IFace != "" {
				defaultConfig.IFace = fileConfig.IFace
			}
			if fileConfig.Discovery != 0 {
				defaultConfig.Discovery = fileConfig.Discovery
			}
			if fileConfig.Port != 0 {
				defaultConfig.Port = fileConfig.Port
			}
		}
	}

	flag.StringVar(&config.Node, "node", defaultConfig.Node, "Name of this node")
	flag.StringVar(&config.IFace, "iface", defaultConfig.IFace, "Network Interface to bind to")
	flag.IntVar(&config.Discovery, "discovery", defaultConfig.Discovery, "Port for network discovery")
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

	return &config
}
