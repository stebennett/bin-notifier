package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
