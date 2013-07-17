package main

import (
	"flag"
	"log"

	"github.com/eikeon/ginger/web"
)

func main() {
	address := flag.String("address", ":9999", "http service address")
	flag.Parse()

	log.Println("starting server on:", *address)

	web.Server(*address)
}
