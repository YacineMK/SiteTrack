package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	mux  *chi.Mux
	http *http.Server
}

func New(port string) *Server {
	r := chi.NewRouter()
	return &Server{
		mux: r,
		http: &http.Server{
			Addr:    port,
			Handler: r,
		},
	}
}

func (s *Server) start() error {
	return s.http.ListenAndServe()
}
