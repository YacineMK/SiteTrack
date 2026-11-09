package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
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
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

func (s *Server) Mux() *chi.Mux {
	return s.mux
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
