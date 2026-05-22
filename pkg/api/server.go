package api

import (
	"errors"
	"net/http"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/store"
)

type Options struct {
	Config     config.Config
	Store      *store.Store
	ReadToken  string
	WriteToken string
}

type Server struct {
	opts Options
	mux  *http.ServeMux
}

func NewServer(opts Options) (*Server, error) {
	if opts.Store == nil {
		return nil, errors.New("store is required")
	}
	if opts.ReadToken == "" || opts.WriteToken == "" {
		return nil, errors.New("read and write tokens are required")
	}
	s := &Server{opts: opts, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	read := RequireToken(s.opts.ReadToken)
	write := RequireToken(s.opts.WriteToken)

	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.Handle("GET /v1/locations", read(http.HandlerFunc(s.handleListLocations)))
	s.mux.Handle("GET /v1/locations/{label}/collections", read(http.HandlerFunc(s.handleListCollections)))
	s.mux.Handle("GET /v1/locations/{label}/collections/next", read(http.HandlerFunc(s.handleNextCollection)))
	s.mux.Handle("PUT /v1/locations/{label}/collections", write(http.HandlerFunc(s.handlePutCollections)))
}
