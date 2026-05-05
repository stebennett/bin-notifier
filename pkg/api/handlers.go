package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.opts.Store.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "unhealthy", err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type locationDTO struct {
	Label    string `json:"label"`
	Postcode string `json:"postcode"`
}

func (s *Server) handleListLocations(w http.ResponseWriter, r *http.Request) {
	out := make([]locationDTO, 0, len(s.opts.Config.Locations))
	for _, loc := range s.opts.Config.Locations {
		out = append(out, locationDTO{Label: loc.Label, Postcode: loc.PostCode})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "")
}
func (s *Server) handleNextCollection(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "")
}
func (s *Server) handlePutCollections(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "")
}
