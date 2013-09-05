package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/eikeon/ginger"
	"github.com/stathat/go"
)

func readUrls(urls chan string) {
	bio := bufio.NewReaderSize(os.Stdin, 50000)
	for {
		line, isPrefix, err := bio.ReadLine()
		if isPrefix {
			log.Println("uhoh, line too long for buffer ", line)
		} else if err == nil {
			cols := strings.Split(string(line), "\t")
			if len(cols) == 3 {
				u := cols[2]
				if u, err := url.Parse(URL); err == nil {
					urls <- u
				} else {
					log.Println("Could not parse:", u)
				}
			}
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
	count := 0
	for url := range urls {
		count += 1
		if err := g.Add(url, name); err != nil {
			log.Println("error:", err)
		}
		if count%1000 == 0 {
			stathat.PostEZCount("gingerload", "eikeon@eikeon.com", 1000)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	C := 64

	urls := make(chan string, 65536)
	hostname, _ := os.Hostname()

	go readUrls(urls)

	var wg sync.WaitGroup
	for i := 0; i < C; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendUrls(urls, fmt.Sprintf("%s-%d", hostname, i))
		}()
	}
	wg.Wait()
}
