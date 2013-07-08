package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("started")

	// We must use a buffered channel or risk missing the signal
	notifyChannel := make(chan os.Signal, 1)

	signal.Notify(notifyChannel, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	sig := <-notifyChannel
	log.Println("stopped:", sig)

}
