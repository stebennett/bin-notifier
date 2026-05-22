package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
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

// parseFromAndTypes returns the `from` query param (defaulting to today in
// Europe/London) and the repeatable `type` filter values.
func parseFromAndTypes(r *http.Request) (string, []string) {
	from := r.URL.Query().Get("from")
	if from == "" {
		from = dateutil.TodayString()
	}
	return from, r.URL.Query()["type"]
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	if !s.knownLocation(label) {
		writeError(w, http.StatusNotFound, "unknown_location", "no such location: "+label)
		return
	}

	from, types := parseFromAndTypes(r)

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

type nextResponse struct {
	Location  string   `json:"location"`
	ScrapedAt string   `json:"scraped_at"`
	Date      string   `json:"date"`
	BinTypes  []string `json:"bin_types"`
}

func (s *Server) handleNextCollection(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	if !s.knownLocation(label) {
		writeError(w, http.StatusNotFound, "unknown_location", "no such location: "+label)
		return
	}

	from, types := parseFromAndTypes(r)

	date, binTypes, scrapedAt, err := s.opts.Store.NextCollection(label, from, types)
	switch {
	case errors.Is(err, store.ErrNoData):
		writeError(w, http.StatusServiceUnavailable, "no_data", "no data cached for location "+label)
		return
	case errors.Is(err, store.ErrNoMatch):
		writeError(w, http.StatusNotFound, "no_match", "no matching collection")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nextResponse{
		Location:  label,
		ScrapedAt: scrapedAt.UTC().Format(time.RFC3339),
		Date:      date,
		BinTypes:  binTypes,
	})
}

type putRequest struct {
	ScrapedAt   string             `json:"scraped_at"`
	Collections []store.Collection `json:"collections"`
}

func (s *Server) handlePutCollections(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	if !s.knownLocation(label) {
		writeError(w, http.StatusNotFound, "unknown_location", "no such location: "+label)
		return
	}

	var body putRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON: "+err.Error())
		return
	}
	scrapedAt, err := time.Parse(time.RFC3339, body.ScrapedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid scraped_at: "+err.Error())
		return
	}
	for _, c := range body.Collections {
		if c.BinType == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "collection.bin_type is required")
			return
		}
		if _, err := time.Parse("2006-01-02", c.Date); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid collection.date: "+c.Date)
			return
		}
	}

	if err := s.opts.Store.ReplaceCollections(label, scrapedAt, body.Collections); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
