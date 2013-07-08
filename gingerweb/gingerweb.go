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
	for {
		select {
		case sig := <-notifyChannel:
			switch sig {
			case os.Interrupt:
				log.Println("handling:", sig)
				goto Done
			default:
				log.Fatal("Unexpected Signal:", sig)
			}
		}
	}
Done:
	log.Println("stopped")

}
