package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/eikeon/ginger"
)

func readUrls(urls chan string) {
	bio := bufio.NewReaderSize(os.Stdin, 50000)
	for {
		line, isPrefix, err := bio.ReadLine()
		if isPrefix {
			log.Println("uhoh, line too long for buffer ", line)
		} else if err == nil {
			cols := strings.Split(string(line), "\t")
			urls <- cols[2]
		} else if err == io.EOF {
			break
		} else {
			panic(err)
		}
	}
	close(urls)
}

func sendUrls(urls chan string, name string) {
	g := ginger.NewMemoryGinger(true)
	for url := range urls {
		g.Add(url, name)
	}
}

func main() {
	urls := make(chan string)
	hostname, _ := os.Hostname()

	go readUrls(urls)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendUrls(urls, fmt.Sprintf("%s-%d", hostname, i))
		}()
	}
	wg.Wait()
}
