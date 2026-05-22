package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := `from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AA"
    address_code: "100000000000"
    collection_days:
      - day: monday
        types: [General Waste]
`
	p := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte(cfg), 0o600))
	return p
}

func TestServerStartsAndServesHealthz(t *testing.T) {
	cfgPath := writeConfig(t)
	dbPath := filepath.Join(t.TempDir(), "cache.db")

	t.Setenv("BN_API_READ_TOKEN", "r")
	t.Setenv("BN_API_WRITE_TOKEN", "w")
	t.Setenv("BN_API_CONFIG_FILE", cfgPath)
	t.Setenv("BN_API_DB_PATH", dbPath)
	t.Setenv("BN_API_LISTEN_ADDR", "127.0.0.1:0")

	app, err := newApp()
	require.NoError(t, err)
	defer app.Close()

	go func() { _ = app.ServeOn(app.Listener) }()

	addr := app.Listener.Addr().String()
	deadline := time.Now().Add(2 * time.Second)
	var resp *http.Response
	for time.Now().Before(deadline) {
		r, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			resp = r
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.NotNil(t, resp)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.True(t, strings.Contains(string(body), "ok"))
}
