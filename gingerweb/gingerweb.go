package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"code.google.com/p/go.net/websocket"

	"github.com/eikeon/ginger"
	"github.com/eikeon/ginger/db"
	"github.com/eikeon/ginger/queue"
)

var site = template.Must(template.ParseFiles("templates/site.html"))

func makeTemplate(names ...string) *template.Template {
	t, err := site.Clone()
	if err != nil {
		log.Fatal("cloning site: ", err)
	}
	return template.Must(t.ParseFiles(names...))
}

type Data map[string]interface{}

func HandleTemplate(prefix, name string, data Data) {
	t := makeTemplate("templates/" + name + ".html")
	http.HandleFunc(prefix, func(w http.ResponseWriter, req *http.Request) {
		d := Data{}
		if data != nil {
			for k, v := range data {
				d[k] = v
			}
		}
		if req.URL.Path == prefix {
			d["Found"] = true
		} else {
			w.Header().Set("Cache-Control", "max-age=10, must-revalidate")
			w.WriteHeader(http.StatusNotFound)
		}
		var bw bytes.Buffer
		h := md5.New()
		mw := io.MultiWriter(&bw, h)
		err := t.ExecuteTemplate(mw, "html", d)
		if err == nil {
			w.Header().Set("ETag", fmt.Sprintf(`"%x"`, h.Sum(nil)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", bw.Len()))
			w.Write(bw.Bytes())
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func main() {
	address := flag.String("address", ":9999", "http service address")
	root := flag.String("root", ".", "...")
	flag.Parse()

	log.Println("started")

	requests := queue.NewChannelQueue(nil)
	responses := queue.NewChannelQueue(nil)
	mdb := &db.MemoryDB{}
	mdb.CreateTable("collection", []db.AttributeDefinition{}, db.KeySchema{})
	mdb.CreateTable("fetchresponse", []db.AttributeDefinition{}, db.KeySchema{})
	g := ginger.NewGinger(requests, responses, mdb)

	fs := http.FileServer(http.Dir(path.Join(*root, "static/")))
	http.Handle("/bootstrap/", fs)
	http.Handle("/jquery/", fs)
	http.Handle("/js/", fs)

	HandleTemplate("/", "home", Data{"Ginger": g})
	HandleTemplate("/collection/", "collection", Data{"Ginger": g})
	http.HandleFunc("/collection", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			if err := req.ParseForm(); err == nil {
				name, ok := req.Form["name"]
				if ok {
					if err := g.AddCollection(name[0], req.RemoteAddr); err != nil {
						log.Println("Error adding collection:", err)
					} else {
						http.Redirect(w, req, req.URL.Path+"/", http.StatusCreated)
					}
				}
			}
			// TODO: write a response
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.Handle("/state", websocket.Handler(func(ws *websocket.Conn) {
		go func() {
			for {
				var msg map[string]interface{}
				if err := websocket.JSON.Receive(ws, &msg); err == nil {
					log.Printf("ignoring: %#v\n", msg)
				} else {
					log.Println("State Websocket receive err:", err)
					return
				}
			}
		}()
		for {
			state := struct {
				Collections []ginger.Collection
			}{g.Collections()}
			if err := websocket.JSON.Send(ws, state); err != nil {
				log.Println("State Websocket send err:", err)
				return
			}
			g.WaitStateChanged()
		}
	}))

	go func() {
		log.Println("starting server on:", *address)
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
