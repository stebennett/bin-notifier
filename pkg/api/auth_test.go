package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func assertUnauthorizedJSON(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.Contains(t, rr.Body.String(), `"code":"unauthorized"`)
}

func TestRequireTokenAcceptsCorrectToken(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer read-secret")
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTokenRejectsWrongToken(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer nope")
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	assertUnauthorizedJSON(t, rr)
}

func TestRequireTokenRejectsMissingHeader(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	assertUnauthorizedJSON(t, rr)
}

func TestRequireTokenRejectsMalformedHeader(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "read-secret") // missing "Bearer "
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	assertUnauthorizedJSON(t, rr)
}

func TestRequireTokenEmptyConfiguredTokenAlwaysRejects(t *testing.T) {
	mw := RequireToken("")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	assertUnauthorizedJSON(t, rr)
}
