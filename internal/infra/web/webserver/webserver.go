package webserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

type WebServer struct {
	Router        chi.Router
	Handlers      []route
	WebServerPort string
}

func NewWebServer(serverPort string) *WebServer {
	return &WebServer{
		Router:        chi.NewRouter(),
		WebServerPort: serverPort,
	}
}

func (s *WebServer) AddHandler(method, path string, handler http.HandlerFunc) {
	s.Handlers = append(s.Handlers, route{Method: method, Path: path, Handler: handler})
}

// loop through the handlers and add them to the router
// register middeleware logger
// start the server
func (s *WebServer) Start() {
	s.Router.Use(middleware.Logger)
	for _, r := range s.Handlers {
		s.Router.Method(r.Method, r.Path, r.Handler)
	}
	http.ListenAndServe(s.WebServerPort, s.Router)
}
