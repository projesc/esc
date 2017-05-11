package main

import "github.com/projesc/esc"

func main() {
	esc.Start()

	end := make(chan bool, 1)
	<-end
}
