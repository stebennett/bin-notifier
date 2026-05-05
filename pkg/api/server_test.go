package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/store"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	cfg := config.Config{Locations: []config.Location{{Label: "Home", PostCode: "RG12 1AA"}}}
	srv, err := NewServer(Options{
		Config:     cfg,
		Store:      s,
		ReadToken:  "read-token",
		WriteToken: "write-token",
	})
	require.NoError(t, err)
	return srv
}

func TestHealthzNoAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
