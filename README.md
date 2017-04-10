# Evented Scripting Cluster (name subject to change)

This project is an study and attempt to create a replicated cluster of scripts to fire events and commands on and off.

## Features

- Single binary (easy to deploy)
- Event senders & listeners
- Command senders & listeners
- Synced key-value store
- Script directory sync
- Lua scripting
- Auto discovery of nodes 
- Auto (re)load scripts
- Works on raspberry pi (including zero)

Soon:

- Plugin system
- IPV6 support
- Docker swarm compatible 
- Join mode / non-discovery mode

## Quick start

Download the proper binary from the [releases page](https://github.com/diogok/esc/releases):

    $ wget https://github.com/diogok/esc/releases/download/0.0.1/esc-amd64 -O esc

Create scripts directory:

    $ mkdir scripts

Start the program on proper network interface:

    $ ./esc -iface wlan0

Repeat on all nodes, edit the scripts on any one of them.

## Complete usage

Download the binary from the [releases page](https://github.com/diogok/esc/releases).

It accepts a configuration file or comand line argumetns (or both with the cli override the file), and comes with sane defaults.

To use the config file pass it as last argument of the command.

```yaml
node: "mynode"
discovery: 8902
port: 8901
scripts: "script_dir"
iface: "wlan0"
```

You can view the command options:

    $ ./esc --help

They are all optional:

    $ ./esc --node mynode --port 8901 --discovery 8902 --iface wlan0 --scripts scripts_dir config.yml

Arguments and config are:

- node: Name of this node, defaults to hostname
- discovery: port for auto-discovery of nodes
- port: port for inter node communication
- scripts: script directory
- iface: interface to bind to (eth0, wlan0...)

## Docker

You can run at each node with docker as:

    $ docker run -v /opt/scripts:/scripts -p8901:8901 -p8902:8902 diogok/esc

Or with arguments:

    $ docker run -v /opt/scripts:/scripts -p8901:8901 -p8902:8902 diogok/esc -scripts /scripts

Docker swarm compatibility is planned but not supported right now.

## Discovery, Messaging and Scripting

The project will use mDNS on discovery port to find other nodes, it will connect and disconnect automatically to them.

Any script at the "scripts" directory will be executed and reloaded on changes, it will also be kept in sync between nodes.

An _event_ is a message with a _name_ and a _payload_ sent to all servers, and a _command_ is a message with a _target_, a _name_ and a _payload_.

Right now it handles only strings.

And example of a script exploring the available functions:

```lua
log("I am "..self()) -- this node name, in case you need

-- listen on events from any node (the first "*") that are named hello
onEvent("*","hello",function(msg)
    log("From "..msg.From.." got "..msg.Name..": "..msg.Payload)
    log("And foo is "..Get("foo"))
end)

-- listen on to the command clear from nodeb
onCommand("nodeb","clear",function(msg)
  -- Key value store is shared
  Set("foo","")
end)

onCommand("*","shutdown",function(msg) 
  -- os and all libs are available
   os.execute("shutdown")
end)

-- Send command to nodeb, could be to everyone with "*"
sendCommand("nodeb","blink","led3")

-- Keep a thread going at each 2secs until stopped
tick(2,function()
    i = i + 1
    log("Sending hello "..i)
    sendEvent("hello","world")
    return i <= 5
    -- if return is true it will loop
end)
```

Each lua script is an instance on it's own, and can only communicate using messages.

A nice lua guide is at [learn X in Y minutes](https://learnxinyminutes.com/docs/lua).

## License

MIT

