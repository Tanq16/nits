package mermaidsvg

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
	mux      *http.ServeMux
	staticFS fs.FS
}

func New() (*Server, error) {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:      http.NewServeMux(),
		staticFS: staticFS,
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.staticFS))))
	s.mux.HandleFunc("/", s.handleIndex)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func Run(addr string) error {
	s, err := New()
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("INFO [mermaidsvg] Listening on http://localhost%s", addr)
	return server.ListenAndServe()
}
