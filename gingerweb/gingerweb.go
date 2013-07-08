package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	log.Println("started")

	notifyChannel := make(chan os.Signal)
	signal.Notify(notifyChannel, os.Interrupt)
	<-notifyChannel
	log.Println("stopped")

}
