package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/web"
)

func main() {
	address := flag.String("address", ":9999", "http service address")
	flag.Parse()

	g := ginger.NewMemoryGinger()
	web.AddHandlers(g)

	go func() {
		log.Println("server listening on:", *address)
		err := http.ListenAndServe(*address, nil)
		if err != nil {
			log.Print("ListenAndServe:", err)
		}
	}()

	// We must use a buffered channel or risk missing the signal
	notifyChannel := make(chan os.Signal, 1)

	signal.Notify(notifyChannel, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	sig := <-notifyChannel
	log.Println("stopped:", sig)

}
