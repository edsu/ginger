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
var templates = make(map[string]*template.Template)

func init() {
	if p, err := build.Default.Import("github.com/eikeon/ginger/web", "", build.FindOnly); err == nil {
		Root = p.Dir
	} else {
		log.Println("WARNING: could not import package:", err)
	}
}

func getTemplate(name string) *template.Template {
	if t, ok := templates[name]; ok {
		return t
	} else {
		if site == nil {
			site = template.Must(template.ParseFiles(path.Join(Root, "templates/site.html")))
		}
		t, err := site.Clone()
		if err != nil {
			log.Fatal("cloning site: ", err)
		}
		t = template.Must(t.ParseFiles(path.Join(Root, name)))
		templates[name] = t
		return t
	}
}

type templateData map[string]interface{}

func writeTemplate(t *template.Template, d templateData, w http.ResponseWriter) {
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

func handleTemplate(prefix, name string, data templateData) {
	t := getTemplate("templates/" + name + ".html")
	http.HandleFunc(prefix, func(w http.ResponseWriter, req *http.Request) {
		d := templateData{}
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
		writeTemplate(t, d, w)
	})
}

type collectionHandler struct {
	g *ginger.Ginger
}

func (c *collectionHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		d := templateData{"Ginger": c.g}
		dir, file := path.Split(req.URL.Path)
		name := path.Base(req.URL.Path)
		var t *template.Template
		if dir == "/collection/" && file == "" {
			d["Found"] = true
			t = getTemplate("templates/" + "collections" + ".html")
		} else if name != "" {
			d["Found"] = true
			for _, collection := range c.g.Collections() {
				if collection.Name == name {
					d["Collection"] = collection
				}
			}
			t = getTemplate("templates/" + "collection" + ".html")
		} else {
			w.Header().Set("Cache-Control", "max-age=10, must-revalidate")
			w.WriteHeader(http.StatusNotFound)
			t = site
		}
		writeTemplate(t, d, w)
	} else if req.Method == "POST" {
		if err := req.ParseForm(); err == nil {
			name, ok := req.Form["name"]
			if ok {
				if _, err := c.g.AddCollection(name[0], req.RemoteAddr); err != nil {
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
}

func AddHandlers(g *ginger.Ginger) {
	fs := http.FileServer(http.Dir(path.Join(Root, "static/")))
	http.Handle("/bootstrap/", fs)
	http.Handle("/jquery/", fs)
	http.Handle("/js/", fs)

	handleTemplate("/", "home", templateData{"Ginger": g})

	http.Handle("/collection/", &collectionHandler{g})

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
