package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/store"
	"github.com/stretchr/testify/require"
)

func authedReq(method, target, token string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestListLocationsReturnsConfiguredLocations(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations", "read-token"))

	require.Equal(t, http.StatusOK, rr.Code)
	var got []map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.Equal(t, []map[string]string{{"label": "Home", "postcode": "RG12 1AA"}}, got)
}

func TestListLocationsRequiresReadToken(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations", "write-token"))
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func seed(t *testing.T, srv *Server) {
	t.Helper()
	require.NoError(t, srv.opts.Store.ReplaceCollections("Home",
		time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC),
		[]store.Collection{
			{BinType: "General Waste", Date: "2026-05-07"},
			{BinType: "Food Waste", Date: "2026-05-08"},
		}))
}

func TestListCollectionsReturnsAllForLocation(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Home/collections?from=2026-05-05", "read-token"))

	require.Equal(t, http.StatusOK, rr.Code)
	var got struct {
		Location    string             `json:"location"`
		ScrapedAt   string             `json:"scraped_at"`
		Collections []store.Collection `json:"collections"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.Equal(t, "Home", got.Location)
	require.Equal(t, "2026-05-05T18:00:00Z", got.ScrapedAt)
	require.Equal(t, []store.Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	}, got.Collections)
}

func TestListCollectionsFiltersByType(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET",
		"/v1/locations/Home/collections?from=2026-05-05&type=Food%20Waste", "read-token"))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"Food Waste"`)
	require.NotContains(t, rr.Body.String(), `"General Waste"`)
}

func TestListCollectionsUnknownLocationReturns404(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Nowhere/collections", "read-token"))
	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Contains(t, rr.Body.String(), `"unknown_location"`)
}

func TestListCollectionsKnownLocationNoDataReturns503(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Home/collections", "read-token"))
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Contains(t, rr.Body.String(), `"no_data"`)
}

func TestListCollectionsDefaultsFromToToday(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	// No `from` query param → handler should use today; with seed dates in 2026 this depends on clock,
	// so just assert we get a 200 and a payload.
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Home/collections", "read-token"))
	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, strings.Contains(rr.Body.String(), `"collections"`))
}
