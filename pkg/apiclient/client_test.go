package apiclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPushCollectionsSendsAuthorizedPUT(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		require.Equal(t, "PUT", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "write-token")
	err := c.PushCollections("Home",
		time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC),
		[]Collection{{BinType: "General Waste", Date: "2026-05-07"}})
	require.NoError(t, err)

	require.Equal(t, "/v1/locations/Home/collections", gotPath)
	require.Equal(t, "Bearer write-token", gotAuth)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(gotBody), &body))
	require.Equal(t, "2026-05-05T18:00:00Z", body["scraped_at"])
}

func TestPushCollectionsRetriesOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "t")
	c.RetryDelay = 1 * time.Millisecond
	err := c.PushCollections("Home", time.Now(), nil)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
}

func TestPushCollectionsReturnsErrorAfterRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "t")
	c.RetryDelay = 1 * time.Millisecond
	err := c.PushCollections("Home", time.Now(), nil)
	require.Error(t, err)
}

func TestURLEscapesLocationLabel(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "t")
	require.NoError(t, c.PushCollections("Holiday Home", time.Now(), nil))
	require.Equal(t, "/v1/locations/Holiday%20Home/collections", gotPath)
}
