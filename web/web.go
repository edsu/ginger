package web

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"go/build"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"

	"code.google.com/p/go.net/websocket"
	"github.com/eikeon/ginger"
)

var Root string
var site *template.Template

func init() {
	p, err := build.Default.Import("github.com/eikeon/ginger/web", "", build.FindOnly)
	if err != nil {
		log.Fatal("could not import package:", err)
	}
	Root = p.Dir
}
func makeTemplate(name string) *template.Template {
	if site == nil {
		site = template.Must(template.ParseFiles(path.Join(Root, "templates/site.html")))
	}
	t, err := site.Clone()
	if err != nil {
		log.Fatal("cloning site: ", err)
	}
	return template.Must(t.ParseFiles(path.Join(Root, name)))
}

type Data map[string]interface{}

func WriteTemplate(t *template.Template, d Data, w http.ResponseWriter) {
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
}

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
		WriteTemplate(t, d, w)
	})
}

func AddHandlers(g *ginger.Ginger) {
	fs := http.FileServer(http.Dir(path.Join(Root, "static/")))
	http.Handle("/bootstrap/", fs)
	http.Handle("/jquery/", fs)
	http.Handle("/js/", fs)

	HandleTemplate("/", "home", Data{"Ginger": g})
	data := Data{"Ginger": g}
	collectionsTemplate := makeTemplate("templates/" + "collections" + ".html")
	collectionTemplate := makeTemplate("templates/" + "collection" + ".html")
	http.HandleFunc("/collection/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			d := Data{}
			if data != nil {
				for k, v := range data {
					d[k] = v
				}
			}
			dir, file := path.Split(req.URL.Path)
			name := path.Base(req.URL.Path)
			if dir == "/collection/" && file == "" {
				d["Found"] = true
				WriteTemplate(collectionsTemplate, d, w)
			} else if name != "" {
				d["Found"] = true
				for _, collection := range g.Collections() {
					if collection.Name == name {
						d["Collection"] = collection
					}
				}
				WriteTemplate(collectionTemplate, d, w)
			} else {
				w.Header().Set("Cache-Control", "max-age=10, must-revalidate")
				w.WriteHeader(http.StatusNotFound)
			}
		} else if req.Method == "POST" {
			if err := req.ParseForm(); err == nil {
				name, ok := req.Form["name"]
				if ok {
					if _, err := g.AddCollection(name[0], req.RemoteAddr); err != nil {
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

}
