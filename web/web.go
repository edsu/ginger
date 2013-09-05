package web

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
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

type gingerServer struct {
	g *ginger.Ginger
}

func (cs *gingerServer) GingerServer(ws *websocket.Conn) {
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
			Fetched []*ginger.Fetch
		}{}
		if cs.g != nil {
			state.Fetched = cs.g.Fetched()
		}
		if err := websocket.JSON.Send(ws, state); err != nil {
			log.Println("State Websocket send err:", err)
			return
		}
		cs.g.WaitStateChanged()
	}
}

func (cs *gingerServer) Add(ws *websocket.Conn) {
	for {

		var url string
		websocket.Message.Receive(ws, &url)

		message := "added:" + url

		if err := cs.g.Add(url, ws.RemoteAddr().String()); err != nil {
			message = fmt.Sprintf("error:", err)
		}

		websocket.Message.Send(ws, message)
	}
}

type gingerHandler struct {
	g *ginger.Ginger
}

func (ch *gingerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var rest string
	var urlregex = regexp.MustCompile(`^/(?P<rest>[^/]+)?$`)
	matches := urlregex.FindStringSubmatch(req.URL.Path)
	if len(matches) > 0 {
		rest = matches[1]
	}
	var t *template.Template
	d := templateData{"Ginger": ch.g}

	d["Ginger"] = ch.g
	if rest == "" {
		if req.Method == "GET" {
			t = getTemplate("templates/" + "home" + ".html")
			d["Found"] = true
		} else if req.Method == "POST" {
			dec := json.NewDecoder(req.Body)
			var v map[string]interface{}
			if err := dec.Decode(&v); err == nil {
				url, ok := v["url"].(string)
				if ok {
					if err := ch.g.Add(url, req.RemoteAddr); err != nil {
						log.Println("Error adding URL:", err)
					} else {
						ch.g.StateChanged()
						http.Redirect(w, req, req.URL.Path+"/", http.StatusCreated)
					}
				} else {
					log.Printf("not string as expected: %#v, %T\n", v["url"], v["url"])
				}
			} else {
				log.Println("couldn't add URL:", err)
			}
			req.Body.Close()
			// TODO: write a response
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else if rest == "state" {
		cs := &gingerServer{ch.g}
		websocket.Handler(cs.GingerServer).ServeHTTP(w, req)
	} else if rest == "add" {
		cs := &gingerServer{ch.g}
		websocket.Handler(cs.Add).ServeHTTP(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
		t = getTemplate("templates/" + "home" + ".html")
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
	http.Handle("/css/", fs)

	http.Handle("/", &gingerHandler{g})
}
