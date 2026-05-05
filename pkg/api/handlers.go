package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/stebennett/bin-notifier/pkg/store"
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

type collectionsResponse struct {
	Location    string             `json:"location"`
	ScrapedAt   string             `json:"scraped_at"`
	Collections []store.Collection `json:"collections"`
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	if !s.knownLocation(label) {
		writeError(w, http.StatusNotFound, "unknown_location", "no such location: "+label)
		return
	}

	from := r.URL.Query().Get("from")
	if from == "" {
		from = time.Now().UTC().Format("2006-01-02")
	}
	types := r.URL.Query()["type"]

	rows, scrapedAt, err := s.opts.Store.ListCollections(label, from, types)
	if errors.Is(err, store.ErrNoData) {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no data cached for location "+label)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	resp := collectionsResponse{
		Location:    label,
		ScrapedAt:   scrapedAt.UTC().Format(time.RFC3339),
		Collections: rows,
	}
	if resp.Collections == nil {
		resp.Collections = []store.Collection{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) knownLocation(label string) bool {
	for _, loc := range s.opts.Config.Locations {
		if loc.Label == label {
			return true
		}
	}
	return false
}
func (s *Server) handleNextCollection(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "")
}
func (s *Server) handlePutCollections(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "")
}
