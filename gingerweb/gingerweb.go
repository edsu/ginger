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

	"github.com/eikeon/ginger"
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

	g := &ginger.Ginger{}

	HandleTemplate("/", "home", Data{"Ginger": g})

	fs := http.FileServer(http.Dir(path.Join(*root, "static/")))
	http.Handle("/bootstrap/", fs)
	http.Handle("/jquery/", fs)
	http.Handle("/js/", fs)

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
