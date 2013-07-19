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
	"regexp"

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

type collectionsServer struct {
	g *ginger.Ginger
}

func (cs *collectionsServer) CollectionsServer(ws *websocket.Conn) {
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
		}{cs.g.Collections()}
		if err := websocket.JSON.Send(ws, state); err != nil {
			log.Println("State Websocket send err:", err)
			return
		}
		cs.g.WaitStateChanged()
	}
}

type collectionServer struct {
	g              *ginger.Ginger
	collectionName string
}

func (cs *collectionServer) CollectionServer(ws *websocket.Conn) {
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
			Fetches []ginger.Fetch
		}{}
		c, _ := cs.g.GetCollection(cs.collectionName)
		if c != nil {
			state.Fetches = c.Fetches()
		}
		if err := websocket.JSON.Send(ws, state); err != nil {
			log.Println("State Websocket send err:", err)
			return
		}
		cs.g.WaitStateChanged()
	}
}

type collectionHandler struct {
	g *ginger.Ginger
}

func (ch *collectionHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var name, rest string
	var urlregex = regexp.MustCompile(`^/collection/(?:(?P<name>[^/]+)/)?(?P<rest>[^/]+)?$`)
	matches := urlregex.FindStringSubmatch(req.URL.Path)
	if len(matches) > 0 {
		name = matches[1]
		rest = matches[2]
	}
	var t *template.Template
	d := templateData{"Ginger": ch.g}
	if name == "" { // collections
		if rest == "" {
			if req.Method == "GET" {
				d["Found"] = true
				t = getTemplate("templates/" + "collections" + ".html")
			} else if req.Method == "POST" {
				if err := req.ParseForm(); err == nil {
					name, ok := req.Form["name"]
					if ok {
						if _, err := ch.g.AddCollection(name[0], req.RemoteAddr); err != nil {
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
		} else if rest == "state" {
			cs := &collectionsServer{ch.g}
			websocket.Handler(cs.CollectionsServer).ServeHTTP(w, req)
		} else {
			w.WriteHeader(http.StatusNotFound)
			t = getTemplate("templates/" + "collections" + ".html")
		}
	} else { // collection named name
		collection, _ := ch.g.GetCollection(name)
		d["Collection"] = collection
		if rest == "" {
			if collection == nil {
				w.WriteHeader(http.StatusNotFound)
				t = getTemplate("templates/" + "collection" + ".html")
			} else if req.Method == "GET" {
				t = getTemplate("templates/" + "collection" + ".html")
				d["Found"] = true
			} else if req.Method == "POST" {
				if err := req.ParseForm(); err == nil {
					url, ok := req.Form["url"]
					if ok {
						if err := collection.Add(url[0], req.RemoteAddr); err != nil {
							log.Println("Error adding URL:", err)
						} else {
							ch.g.StateChanged()
							http.Redirect(w, req, req.URL.Path+"/", http.StatusCreated)
						}
					}
				}
				// TODO: write a response
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else if rest == "state" {
			cs := &collectionServer{ch.g, name}
			websocket.Handler(cs.CollectionServer).ServeHTTP(w, req)
		} else {
			w.WriteHeader(http.StatusNotFound)
			t = getTemplate("templates/" + "collection" + ".html")
		}
	}
	if t != nil {
		writeTemplate(t, d, w)
	}

}

func AddHandlers(g *ginger.Ginger) {
	fs := http.FileServer(http.Dir(path.Join(Root, "static/")))
	http.Handle("/bootstrap/", fs)
	http.Handle("/jquery/", fs)
	http.Handle("/js/", fs)

	handleTemplate("/", "home", templateData{"Ginger": g})

	http.Handle("/collection/", &collectionHandler{g})
}
