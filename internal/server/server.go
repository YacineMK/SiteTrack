package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	mux  *chi.Mux
	http *http.Server
}

func New(addr string) *Server {
	r := chi.NewRouter()
	return &Server{
		mux: r,
		http: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}

func (s *Server) Mux() *chi.Mux {
	return s.mux
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}
