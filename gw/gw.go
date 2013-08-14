package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/queue"
	"github.com/eikeon/ginger/web"
)

// run other bits of stack for testing purposes
func testing(g *ginger.Ginger) {
	requests := queue.NewChannelQueue(nil)

	go func() {
		for {
			ginger.Qer(requests)
			time.Sleep(10 * time.Second)
		}
	}()

	go ginger.Worker(requests)

	go func() {
		for {
			g.StateChanged()
			time.Sleep(1 * time.Second)
		}
	}()
}

func main() {
	address := flag.String("address", ":9999", "http service address")
	dynamo := flag.Bool("dynamo", false, "use dynamodb")
	flag.Parse()

	g := ginger.NewMemoryGinger(*dynamo)
	web.AddHandlers(g)

	testing(g)

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
