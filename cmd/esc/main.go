package main

import "esc"

func main() {
	esc.Start()

	end := make(chan bool, 1)
	<-end
}
