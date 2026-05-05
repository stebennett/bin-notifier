# bin-notifier HTTP API and Python MCP — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing Go MCP server with an HTTP API (fed by the existing nightly notifier) and a new Python/uv MCP server that consumes the API.

**Architecture:** One new long-running Go HTTP server (`cmd/api/`) owns a SQLite cache. The existing notifier (`cmd/notifier/`) PUTs scraped data to the API after each scrape, before the SMS step. A new Python FastMCP server (`mcp/`) calls the API. The old Go MCP server (`cmd/server/`, `Dockerfile.mcp`) is deleted. Everything is packaged for k3s via a Helm chart at `deploy/helm/bin-notifier/`.

**Tech Stack:** Go 1.22+ (`net/http`, `modernc.org/sqlite`, `testify`); Python 3.12 with uv, FastMCP (`mcp[cli]`), `httpx`, `pytest`, `respx`; Helm 3.

**Spec:** `docs/superpowers/specs/2026-05-05-bin-notifier-api-and-python-mcp-design.md`

---

## File Map

**New (Go API):**
- `cmd/api/main.go` — entry point (flag parsing, config load, store init, server start)
- `cmd/api/main_test.go` — startup smoke test
- `pkg/store/store.go` — SQLite-backed `Store` with `ReplaceCollections`, `ListCollections`, `NextCollection`
- `pkg/store/store_test.go`
- `pkg/api/server.go` — HTTP server constructor, route registration, request/response types
- `pkg/api/server_test.go`
- `pkg/api/auth.go` — bearer-token middleware (read/write roles)
- `pkg/api/auth_test.go`
- `pkg/api/handlers.go` — handler funcs (locations, collections, next, put)
- `pkg/api/handlers_test.go`
- `pkg/apiclient/client.go` — Go HTTP client used by notifier
- `pkg/apiclient/client_test.go`
- `Dockerfile.api`

**Modified (Go):**
- `cmd/notifier/main.go` — call `apiclient.PushCollections` after each successful scrape, before SMS
- `cmd/notifier/main_test.go` — add a fake API client; verify push happens, push failure does not block SMS
- `go.mod` / `go.sum`
- `.github/workflows/ci.yml` — build `cmd/api`, drop `cmd/server`
- `.github/workflows/release.yml` — release `bin-notifier-api` binary + Docker image, drop MCP
- `CLAUDE.md`, `README.md`

**Deleted:**
- `cmd/server/` (whole directory)
- `Dockerfile.mcp`

**New (Python MCP):**
- `mcp/pyproject.toml`
- `mcp/uv.lock` (generated)
- `mcp/src/bin_notifier_mcp/__init__.py`
- `mcp/src/bin_notifier_mcp/__main__.py`
- `mcp/src/bin_notifier_mcp/server.py` (FastMCP server + tool definitions)
- `mcp/src/bin_notifier_mcp/client.py` (httpx wrapper)
- `mcp/tests/test_server.py`
- `mcp/tests/test_client.py`
- `mcp/Dockerfile`
- `mcp/README.md`

**New (Helm):**
- `deploy/helm/bin-notifier/Chart.yaml`
- `deploy/helm/bin-notifier/values.yaml`
- `deploy/helm/bin-notifier/templates/_helpers.tpl`
- `deploy/helm/bin-notifier/templates/secret-config.yaml`
- `deploy/helm/bin-notifier/templates/secret-tokens.yaml`
- `deploy/helm/bin-notifier/templates/api-deployment.yaml`
- `deploy/helm/bin-notifier/templates/api-service.yaml`
- `deploy/helm/bin-notifier/templates/api-ingress.yaml`
- `deploy/helm/bin-notifier/templates/api-pvc.yaml`
- `deploy/helm/bin-notifier/templates/notifier-cronjob.yaml`
- `deploy/helm/bin-notifier/templates/NOTES.txt`
- `deploy/helm/bin-notifier/.helmignore`

---

## Wire Format (used across many tasks)

JSON shapes used by both the Go API and the Python MCP. Treat these as authoritative.

```json
// GET /v1/locations
[
  {"label": "Home", "postcode": "RG12 1AA"}
]

// GET /v1/locations/{label}/collections
{
  "location": "Home",
  "scraped_at": "2026-05-05T18:00:00Z",
  "collections": [
    {"bin_type": "General Waste", "date": "2026-05-07"},
    {"bin_type": "Food Waste",    "date": "2026-05-08"}
  ]
}

// GET /v1/locations/{label}/collections/next
{
  "location": "Home",
  "scraped_at": "2026-05-05T18:00:00Z",
  "date": "2026-05-07",
  "bin_types": ["General Waste"]
}

// PUT /v1/locations/{label}/collections (request body)
{
  "scraped_at": "2026-05-05T18:00:00Z",
  "collections": [
    {"bin_type": "General Waste", "date": "2026-05-07"}
  ]
}

// Error envelope (any non-2xx)
{"error": "no data cached for location Home", "code": "no_data"}
```

Status codes: 200 success, 204 PUT success, 400 bad request, 401 missing/wrong token, 404 unknown location *or* `/next` had no match, 503 known location with no cached rows.

---

## Task 1: Delete the old Go MCP server

**Files:**
- Delete: `cmd/server/main.go`, `cmd/server/main_test.go`
- Delete: `Dockerfile.mcp`
- Modify: `.github/workflows/ci.yml` (remove `cmd/server` build step)
- Modify: `.github/workflows/release.yml` (remove MCP server build/push steps)
- Modify: `CLAUDE.md` (remove "MCP Server (cmd/server)" section and related docker commands)
- Modify: `README.md` (remove MCP-related sections)

- [ ] **Step 1: Remove the Go MCP code and Dockerfile**

```bash
git rm -r cmd/server
git rm Dockerfile.mcp
```

- [ ] **Step 2: Edit `.github/workflows/ci.yml`**

Open the file, find any `go build` step that references `./cmd/server` and delete it. Keep the `./cmd/notifier` step. Also delete any test step that names `cmd/server` specifically (the broad `go test ./...` step stays).

- [ ] **Step 3: Edit `.github/workflows/release.yml`**

Remove every step or matrix entry that builds, tags, or pushes `bin-notifier-mcp`. Leave the `bin-notifier` notifier release flow intact.

- [ ] **Step 4: Edit `CLAUDE.md`**

Delete:
- The "Build the MCP server", "Run the MCP server", and "Build MCP server Docker image" / "Run MCP server with Docker" blocks under "Build and Run Commands".
- The `cmd/server/` entry under Project Structure.
- The "MCP Server (cmd/server)" section.
- The `mcp-go` line under Key Dependencies.

- [ ] **Step 5: Edit `README.md`**

Search for "MCP" and remove any user-facing instructions for the old MCP server. Leave a single-line note that an MCP server now lives under `mcp/` and will be documented separately.

- [ ] **Step 6: Verify the build still works**

Run: `go build ./... && go test ./...`
Expected: builds and all remaining tests pass.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: remove Go MCP server in favour of upcoming Python MCP"
```

---

## Task 2: Add SQLite store — schema + open

**Files:**
- Create: `pkg/store/store.go`
- Create: `pkg/store/store_test.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the SQLite driver dependency**

Run: `go get modernc.org/sqlite`
Expected: `go.mod` updated with `modernc.org/sqlite` and `go.sum` populated.

- [ ] **Step 2: Write the failing test for `Open` and schema creation**

Create `pkg/store/store_test.go`:

```go
package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	// Re-opening an existing file must succeed (schema migration is idempotent).
	s2, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s2.Close())
}

func TestOpenInMemory(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, s.Close())
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./pkg/store/...`
Expected: FAIL — `Open` undefined.

- [ ] **Step 4: Implement `Store`, `Open`, and schema**

Create `pkg/store/store.go`:

```go
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS collections (
    location    TEXT NOT NULL,
    bin_type    TEXT NOT NULL,
    date        TEXT NOT NULL,
    scraped_at  TEXT NOT NULL,
    PRIMARY KEY (location, bin_type, date)
);
CREATE INDEX IF NOT EXISTS idx_collections_location_date ON collections(location, date);
`

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the DB is reachable; used by /healthz.
func (s *Store) Ping() error { return s.db.Ping() }
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/store/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum pkg/store
git commit -m "feat(store): add SQLite store with schema bootstrap"
```

---

## Task 3: Store — `ReplaceCollections`

**Files:**
- Modify: `pkg/store/store.go`
- Modify: `pkg/store/store_test.go`

- [ ] **Step 1: Write the failing test**

Append to `pkg/store/store_test.go`:

```go
import "time"

func TestReplaceCollectionsInsertsRows(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	err = s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	})
	require.NoError(t, err)

	got, gotScrapedAt, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, scrapedAt.UTC(), gotScrapedAt.UTC())
	require.Equal(t, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	}, got)
}

func TestReplaceCollectionsReplacesExistingLocation(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "Food Waste", Date: "2026-05-08"},
	}))

	got, _, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, []Collection{
		{BinType: "Food Waste", Date: "2026-05-08"},
	}, got)
}

func TestReplaceCollectionsEmptyClearsLocation(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, nil))

	got, _, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestReplaceCollectionsDoesNotAffectOtherLocations(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Office", scrapedAt, []Collection{
		{BinType: "Recycling", Date: "2026-05-09"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, nil))

	got, _, err := s.ListCollections("Office", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, []Collection{{BinType: "Recycling", Date: "2026-05-09"}}, got)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/store/...`
Expected: FAIL — `Collection`, `ReplaceCollections`, `ListCollections` undefined.

- [ ] **Step 3: Implement `Collection`, `ReplaceCollections`, `ListCollections` (skeleton — full ListCollections in next task)**

Append to `pkg/store/store.go`:

```go
import (
	"errors"
	"sort"
	"time"
)

type Collection struct {
	BinType string `json:"bin_type"`
	Date    string `json:"date"` // YYYY-MM-DD
}

var ErrNoData = errors.New("no data for location")

func (s *Store) ReplaceCollections(location string, scrapedAt time.Time, items []Collection) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM collections WHERE location = ?`, location); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO collections (location, bin_type, date, scraped_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scrapedStr := scrapedAt.UTC().Format(time.RFC3339)
	for _, c := range items {
		if _, err := stmt.Exec(location, c.BinType, c.Date, scrapedStr); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListCollections returns rows for location with date >= from, optionally filtered by bin types.
// Returns ErrNoData if there are zero rows for the location at all.
func (s *Store) ListCollections(location string, from string, types []string) ([]Collection, time.Time, error) {
	var anyExists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM collections WHERE location = ?)`, location).Scan(&anyExists); err != nil {
		return nil, time.Time{}, err
	}
	if !anyExists {
		return nil, time.Time{}, ErrNoData
	}

	args := []any{location, from}
	q := `SELECT bin_type, date, scraped_at FROM collections WHERE location = ? AND date >= ?`
	if len(types) > 0 {
		q += ` AND bin_type IN (?` + repeatPlaceholders(len(types)-1) + `)`
		for _, t := range types {
			args = append(args, t)
		}
	}
	q += ` ORDER BY date ASC, bin_type ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()

	var out []Collection
	var latestScraped time.Time
	for rows.Next() {
		var c Collection
		var scrapedStr string
		if err := rows.Scan(&c.BinType, &c.Date, &scrapedStr); err != nil {
			return nil, time.Time{}, err
		}
		ts, err := time.Parse(time.RFC3339, scrapedStr)
		if err != nil {
			return nil, time.Time{}, err
		}
		if ts.After(latestScraped) {
			latestScraped = ts
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, err
	}
	if latestScraped.IsZero() {
		// Location exists but no rows match the filters — fetch any scraped_at for context.
		var scrapedStr string
		if err := s.db.QueryRow(`SELECT scraped_at FROM collections WHERE location = ? LIMIT 1`, location).Scan(&scrapedStr); err == nil {
			if ts, perr := time.Parse(time.RFC3339, scrapedStr); perr == nil {
				latestScraped = ts
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].BinType < out[j].BinType
	})
	return out, latestScraped, nil
}

func repeatPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += ", ?"
	}
	return s
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/store/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/store
git commit -m "feat(store): replace and list collections"
```

---

## Task 4: Store — `NextCollection` and filter tests

**Files:**
- Modify: `pkg/store/store.go`
- Modify: `pkg/store/store_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `pkg/store/store_test.go`:

```go
func TestNextCollectionReturnsEarliestDate(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "Food Waste", Date: "2026-05-08"},
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Recycling", Date: "2026-05-07"},
	}))

	date, types, scraped, err := s.NextCollection("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, "2026-05-07", date)
	require.ElementsMatch(t, []string{"General Waste", "Recycling"}, types)
	require.Equal(t, scrapedAt.UTC(), scraped.UTC())
}

func TestNextCollectionFiltersByType(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	}))

	date, types, _, err := s.NextCollection("Home", "2026-05-05", []string{"Food Waste"})
	require.NoError(t, err)
	require.Equal(t, "2026-05-08", date)
	require.Equal(t, []string{"Food Waste"}, types)
}

func TestNextCollectionNoMatchReturnsErrNoMatch(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))

	_, _, _, err = s.NextCollection("Home", "2026-05-05", []string{"Garden"})
	require.ErrorIs(t, err, ErrNoMatch)
}

func TestNextCollectionUnknownLocationReturnsErrNoData(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	_, _, _, err = s.NextCollection("Nowhere", "2026-05-05", nil)
	require.ErrorIs(t, err, ErrNoData)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/store/...`
Expected: FAIL — `NextCollection`, `ErrNoMatch` undefined.

- [ ] **Step 3: Implement `NextCollection` and `ErrNoMatch`**

Append to `pkg/store/store.go`:

```go
var ErrNoMatch = errors.New("no matching collection")

func (s *Store) NextCollection(location string, from string, types []string) (string, []string, time.Time, error) {
	rows, scrapedAt, err := s.ListCollections(location, from, types)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if len(rows) == 0 {
		return "", nil, time.Time{}, ErrNoMatch
	}
	earliest := rows[0].Date
	var binTypes []string
	for _, r := range rows {
		if r.Date != earliest {
			break
		}
		binTypes = append(binTypes, r.BinType)
	}
	return earliest, binTypes, scrapedAt, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/store/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/store
git commit -m "feat(store): NextCollection with type filter"
```

---

## Task 5: Auth middleware

**Files:**
- Create: `pkg/api/auth.go`
- Create: `pkg/api/auth_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/api/auth_test.go`:

```go
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
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Contains(t, rr.Body.String(), `"code":"unauthorized"`)
}

func TestRequireTokenRejectsMissingHeader(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireTokenRejectsMalformedHeader(t *testing.T) {
	mw := RequireToken("read-secret")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "read-secret") // missing "Bearer "
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireTokenEmptyConfiguredTokenAlwaysRejects(t *testing.T) {
	mw := RequireToken("")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/...`
Expected: FAIL — `RequireToken` undefined.

- [ ] **Step 3: Implement middleware and a JSON error helper**

Create `pkg/api/auth.go`:

```go
package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type Middleware func(http.Handler) http.Handler

// RequireToken returns middleware that requires "Authorization: Bearer <expected>".
// An empty expected token rejects all requests (fail-closed) to avoid accidental open access.
func RequireToken(expected string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if expected == "" || !strings.HasPrefix(h, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid Authorization header")
				return
			}
			got := strings.TrimPrefix(h, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: msg, Code: code})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/api/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api/auth.go pkg/api/auth_test.go
git commit -m "feat(api): bearer-token auth middleware"
```

---

## Task 6: API server — `/healthz` and `/v1/locations`

**Files:**
- Create: `pkg/api/server.go`
- Create: `pkg/api/handlers.go`
- Create: `pkg/api/server_test.go`
- Create: `pkg/api/handlers_test.go`

- [ ] **Step 1: Write failing tests for the server constructor and /healthz**

Create `pkg/api/server_test.go`:

```go
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
```

Create `pkg/api/handlers_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/...`
Expected: FAIL — `Server`, `NewServer`, `Options` undefined.

- [ ] **Step 3: Implement server skeleton**

Create `pkg/api/server.go`:

```go
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
	opts    Options
	mux     *http.ServeMux
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
```

Create `pkg/api/handlers.go` with stubs:

```go
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/api/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api
git commit -m "feat(api): server skeleton with /healthz and /v1/locations"
```

---

## Task 7: API — `GET /v1/locations/{label}/collections`

**Files:**
- Modify: `pkg/api/handlers.go`
- Modify: `pkg/api/handlers_test.go`

- [ ] **Step 1: Write failing tests**

Append to `pkg/api/handlers_test.go`:

```go
import (
	"strings"
	"time"

	"github.com/stebennett/bin-notifier/pkg/store"
)

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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/...`
Expected: FAIL — handler returns `not_implemented` / 404 logic missing.

- [ ] **Step 3: Implement `handleListCollections` and a "known location" helper**

Replace `handleListCollections` in `pkg/api/handlers.go`:

```go
import (
	"errors"
	"time"

	"github.com/stebennett/bin-notifier/pkg/store"
)

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
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/api/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api
git commit -m "feat(api): GET /v1/locations/{label}/collections"
```

---

## Task 8: API — `GET /v1/locations/{label}/collections/next`

**Files:**
- Modify: `pkg/api/handlers.go`
- Modify: `pkg/api/handlers_test.go`

- [ ] **Step 1: Write failing tests**

Append to `pkg/api/handlers_test.go`:

```go
func TestNextCollectionReturnsEarliest(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Home/collections/next?from=2026-05-05", "read-token"))

	require.Equal(t, http.StatusOK, rr.Code)
	var got struct {
		Location  string   `json:"location"`
		ScrapedAt string   `json:"scraped_at"`
		Date      string   `json:"date"`
		BinTypes  []string `json:"bin_types"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.Equal(t, "Home", got.Location)
	require.Equal(t, "2026-05-07", got.Date)
	require.Equal(t, []string{"General Waste"}, got.BinTypes)
}

func TestNextCollectionFiltersByType(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET",
		"/v1/locations/Home/collections/next?from=2026-05-05&type=Food%20Waste", "read-token"))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"2026-05-08"`)
}

func TestNextCollectionNoMatchReturns404(t *testing.T) {
	srv := newTestServer(t)
	seed(t, srv)

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET",
		"/v1/locations/Home/collections/next?from=2026-05-05&type=Garden", "read-token"))
	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Contains(t, rr.Body.String(), `"no_match"`)
}

func TestNextCollectionUnknownLocationReturns404(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Nowhere/collections/next", "read-token"))
	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Contains(t, rr.Body.String(), `"unknown_location"`)
}

func TestNextCollectionKnownLocationNoDataReturns503(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, authedReq("GET", "/v1/locations/Home/collections/next", "read-token"))
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Contains(t, rr.Body.String(), `"no_data"`)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/...`
Expected: FAIL.

- [ ] **Step 3: Implement `handleNextCollection`**

Replace `handleNextCollection` in `pkg/api/handlers.go`:

```go
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

	from := r.URL.Query().Get("from")
	if from == "" {
		from = time.Now().UTC().Format("2006-01-02")
	}
	types := r.URL.Query()["type"]

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
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/api/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api
git commit -m "feat(api): GET /v1/locations/{label}/collections/next"
```

---

## Task 9: API — `PUT /v1/locations/{label}/collections`

**Files:**
- Modify: `pkg/api/handlers.go`
- Modify: `pkg/api/handlers_test.go`

- [ ] **Step 1: Write failing tests**

Append to `pkg/api/handlers_test.go`:

```go
import "bytes"

func putReq(target, token, body string) *http.Request {
	req := httptest.NewRequest("PUT", target, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestPutCollectionsReplacesAndReturns204(t *testing.T) {
	srv := newTestServer(t)
	body := `{"scraped_at":"2026-05-05T18:00:00Z","collections":[
		{"bin_type":"General Waste","date":"2026-05-07"}
	]}`
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, putReq("/v1/locations/Home/collections", "write-token", body))
	require.Equal(t, http.StatusNoContent, rr.Code)

	rows, _, err := srv.opts.Store.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "General Waste", rows[0].BinType)
}

func TestPutCollectionsRequiresWriteToken(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, putReq("/v1/locations/Home/collections", "read-token", `{}`))
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestPutCollectionsUnknownLocationReturns404(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, putReq("/v1/locations/Nowhere/collections", "write-token",
		`{"scraped_at":"2026-05-05T18:00:00Z","collections":[]}`))
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPutCollectionsBadJSONReturns400(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, putReq("/v1/locations/Home/collections", "write-token", `not json`))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPutCollectionsBadDateReturns400(t *testing.T) {
	srv := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, putReq("/v1/locations/Home/collections", "write-token",
		`{"scraped_at":"2026-05-05T18:00:00Z","collections":[{"bin_type":"X","date":"not-a-date"}]}`))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/api/...`
Expected: FAIL.

- [ ] **Step 3: Implement `handlePutCollections`**

Replace `handlePutCollections` in `pkg/api/handlers.go`:

```go
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/api/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/api
git commit -m "feat(api): PUT /v1/locations/{label}/collections"
```

---

## Task 10: API binary — `cmd/api/main.go`

**Files:**
- Create: `cmd/api/main.go`
- Create: `cmd/api/main_test.go`

- [ ] **Step 1: Write failing smoke test**

Create `cmd/api/main_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/api/...`
Expected: FAIL — `newApp` undefined.

- [ ] **Step 3: Implement `cmd/api/main.go`**

Create `cmd/api/main.go`:

```go
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stebennett/bin-notifier/pkg/api"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/store"
)

type app struct {
	server   *http.Server
	store    *store.Store
	Listener net.Listener
}

func (a *app) Close() error {
	if a.server != nil {
		_ = a.server.Shutdown(context.Background())
	}
	if a.store != nil {
		return a.store.Close()
	}
	return nil
}

func (a *app) ServeOn(l net.Listener) error {
	if err := a.server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newApp() (*app, error) {
	configPath := envOr("BN_API_CONFIG_FILE", "")
	dbPath := envOr("BN_API_DB_PATH", "/var/lib/bin-notifier/cache.db")
	listenAddr := envOr("BN_API_LISTEN_ADDR", ":8080")
	readToken := os.Getenv("BN_API_READ_TOKEN")
	writeToken := os.Getenv("BN_API_WRITE_TOKEN")

	if configPath == "" {
		return nil, errors.New("BN_API_CONFIG_FILE is required")
	}
	if readToken == "" || writeToken == "" {
		return nil, errors.New("BN_API_READ_TOKEN and BN_API_WRITE_TOKEN are required")
	}

	cfg, err := config.LoadConfigForMCP(configPath)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	srv, err := api.NewServer(api.Options{
		Config: cfg, Store: st, ReadToken: readToken, WriteToken: writeToken,
	})
	if err != nil {
		st.Close()
		return nil, err
	}

	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		st.Close()
		return nil, err
	}
	return &app{server: httpSrv, store: st, Listener: listener}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Allow `-c` as a convenience flag mirroring other binaries.
	fs := flag.NewFlagSet("bin-notifier-api", flag.ContinueOnError)
	cfgFlag := fs.String("c", "", "path to YAML config file")
	_ = fs.Parse(os.Args[1:])
	if *cfgFlag != "" {
		_ = os.Setenv("BN_API_CONFIG_FILE", *cfgFlag)
	}

	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = a.server.Shutdown(context.Background())
	}()

	log.Printf("bin-notifier-api listening on %s", a.Listener.Addr())
	if err := a.ServeOn(a.Listener); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Run test**

Run: `go test ./cmd/api/...`
Expected: PASS.

- [ ] **Step 5: Verify the whole module still builds**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/api
git commit -m "feat(api): bin-notifier-api binary entry point"
```

---

## Task 11: API client used by the notifier

**Files:**
- Create: `pkg/apiclient/client.go`
- Create: `pkg/apiclient/client_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/apiclient/client_test.go`:

```go
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
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "t")
	require.NoError(t, c.PushCollections("Holiday Home", time.Now(), nil))
	require.Equal(t, "/v1/locations/Holiday%20Home/collections", gotPath)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/apiclient/...`
Expected: FAIL — package missing.

- [ ] **Step 3: Implement the client**

Create `pkg/apiclient/client.go`:

```go
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Collection struct {
	BinType string `json:"bin_type"`
	Date    string `json:"date"`
}

type Client struct {
	BaseURL    string
	Token      string
	HTTP       *http.Client
	RetryDelay time.Duration
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		RetryDelay: 500 * time.Millisecond,
	}
}

type pushBody struct {
	ScrapedAt   string       `json:"scraped_at"`
	Collections []Collection `json:"collections"`
}

func (c *Client) PushCollections(label string, scrapedAt time.Time, items []Collection) error {
	if items == nil {
		items = []Collection{}
	}
	body := pushBody{
		ScrapedAt:   scrapedAt.UTC().Format(time.RFC3339),
		Collections: items,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	target := c.BaseURL + "/v1/locations/" + url.PathEscape(label) + "/collections"

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(c.RetryDelay)
		}
		req, err := http.NewRequest(http.MethodPut, target, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("api returned %d: %s", resp.StatusCode, string(respBody))
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusRequestTimeout {
			// Don't retry 4xx other than timeouts.
			return lastErr
		}
	}
	return lastErr
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/apiclient/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/apiclient
git commit -m "feat(apiclient): Go HTTP client for bin-notifier API"
```

---

## Task 12: Wire the notifier to push to the API

**Files:**
- Modify: `cmd/notifier/main.go`
- Modify: `cmd/notifier/main_test.go`

- [ ] **Step 1: Read the existing notifier test setup**

Run: `sed -n '1,80p' cmd/notifier/main_test.go`
Note the existing mock `BinScraper` and `SMSClient` patterns; add a parallel mock for the API client.

- [ ] **Step 2: Write failing tests**

Append to `cmd/notifier/main_test.go`:

```go
type fakeAPIClient struct {
	calls []apiCall
	fail  bool
}
type apiCall struct {
	Label     string
	ScrapedAt time.Time
	Items     []apiclient.Collection
}

func (f *fakeAPIClient) PushCollections(label string, scrapedAt time.Time, items []apiclient.Collection) error {
	f.calls = append(f.calls, apiCall{Label: label, ScrapedAt: scrapedAt, Items: items})
	if f.fail {
		return fmt.Errorf("simulated api failure")
	}
	return nil
}

func TestNotifierPushesScrapedDataBeforeSMS(t *testing.T) {
	tomorrow := "2026-05-06"
	scraperCollect, _ := time.Parse("2006-01-02", tomorrow)
	cfg := config.Config{
		FromNumber: "+1", ToNumber: "+2", TodayDate: "2026-05-05",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "P", AddressCode: "A",
				CollectionDays: []config.CollectionDay{{Day: time.Wednesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
		},
	}
	api := &fakeAPIClient{}
	sms := &fakeSMSClient{}
	scr := &fakeScraper{times: []scraper.BinTime{{Type: "General Waste", CollectionTime: scraperCollect}}}

	n := &Notifier{
		ScraperFactory: func(string) (BinScraper, error) { return scr, nil },
		SMSClient:      sms,
		APIClient:      api,
		Clock:          func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) },
	}
	results := n.Run(cfg)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)
	require.Len(t, api.calls, 1)
	require.Equal(t, "Home", api.calls[0].Label)
	require.Equal(t, []apiclient.Collection{{BinType: "General Waste", Date: tomorrow}}, api.calls[0].Items)
	require.True(t, sms.sent, "SMS should still be sent")
}

func TestNotifierPushFailureDoesNotBlockSMS(t *testing.T) {
	scraperCollect, _ := time.Parse("2006-01-02", "2026-05-06")
	cfg := config.Config{
		FromNumber: "+1", ToNumber: "+2", TodayDate: "2026-05-05",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "P", AddressCode: "A",
				CollectionDays: []config.CollectionDay{{Day: time.Wednesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
		},
	}
	api := &fakeAPIClient{fail: true}
	sms := &fakeSMSClient{}
	scr := &fakeScraper{times: []scraper.BinTime{{Type: "General Waste", CollectionTime: scraperCollect}}}

	n := &Notifier{
		ScraperFactory: func(string) (BinScraper, error) { return scr, nil },
		SMSClient:      sms,
		APIClient:      api,
		Clock:          func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) },
	}
	results := n.Run(cfg)

	require.Len(t, results, 1)
	require.NoError(t, results[0].Error, "push failure must not surface as a notifier error")
	require.True(t, sms.sent, "SMS should still be sent despite push failure")
}
```

If `fakeSMSClient` and `fakeScraper` already exist in the test file, reuse them; otherwise add the obvious skeletons matching the existing patterns. Imports to add at the top of the test file: `apiclient "github.com/stebennett/bin-notifier/pkg/apiclient"`, `fmt`.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./cmd/notifier/...`
Expected: FAIL — `APIClient` field undefined on `Notifier`.

- [ ] **Step 4: Add `APIClient` interface and wire push call**

In `cmd/notifier/main.go`:

a. Add the interface near the other interfaces:

```go
import "github.com/stebennett/bin-notifier/pkg/apiclient"

// APIClient pushes scraped collections to the bin-notifier API.
type APIClient interface {
	PushCollections(label string, scrapedAt time.Time, items []apiclient.Collection) error
}

// noopAPIClient is used when the API is not configured (no BN_API_BASE_URL).
type noopAPIClient struct{}

func (noopAPIClient) PushCollections(string, time.Time, []apiclient.Collection) error { return nil }
```

b. Add `APIClient APIClient` to the `Notifier` struct.

c. In `processLocation`, after the successful `ScrapeBinTimes` call and before the existing tomorrow-comparison loop, push the data:

```go
items := make([]apiclient.Collection, 0, len(binTimes))
for _, bt := range binTimes {
	items = append(items, apiclient.Collection{
		BinType: bt.Type,
		Date:    bt.CollectionTime.Format("2006-01-02"),
	})
}
if err := n.APIClient.PushCollections(loc.Label, n.Clock(), items); err != nil {
	log.Printf("[%s] WARN: API push failed (continuing to SMS): %v", loc.Label, err)
}
```

d. In `main`, construct the API client from env, fall back to noop:

```go
var apiCli APIClient = noopAPIClient{}
if base := os.Getenv("BN_API_BASE_URL"); base != "" {
	apiCli = apiclient.New(base, os.Getenv("BN_API_WRITE_TOKEN"))
}
notifier := &Notifier{
	ScraperFactory: func(name string) (BinScraper, error) { return scraper.NewScraper(name) },
	SMSClient:      &twilioSMSClientAdapter{client: clients.NewTwilioClient()},
	APIClient:      apiCli,
	Clock:          time.Now,
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./cmd/notifier/... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/notifier
git commit -m "feat(notifier): push scraped data to API before SMS"
```

---

## Task 13: Dockerfile for the API

**Files:**
- Create: `Dockerfile.api`

- [ ] **Step 1: Read the existing notifier Dockerfile**

Run: `cat Dockerfile`
Use it as the model. The API image must NOT include Chrome (no scraping) — small `gcr.io/distroless/static` final image is ideal because `modernc.org/sqlite` is pure Go.

- [ ] **Step 2: Write `Dockerfile.api`**

Create `Dockerfile.api`:

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /out/bin-notifier-api ./cmd/api

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/bin-notifier-api /bin-notifier-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bin-notifier-api"]
```

- [ ] **Step 3: Verify the build**

Run: `docker build -f Dockerfile.api -t bin-notifier-api:test .`
Expected: image built successfully.

- [ ] **Step 4: Smoke test the image**

Run:
```bash
docker run --rm \
  -e BN_API_READ_TOKEN=r -e BN_API_WRITE_TOKEN=w \
  -e BN_API_CONFIG_FILE=/config.yaml \
  -e BN_API_DB_PATH=/tmp/cache.db \
  -v "$PWD/config.example.yaml:/config.yaml:ro" \
  -p 8081:8080 bin-notifier-api:test &
sleep 1
curl -fsS http://127.0.0.1:8081/healthz && echo
docker ps --filter ancestor=bin-notifier-api:test -q | xargs -r docker kill
```
Expected: `ok` printed.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile.api
git commit -m "build: Dockerfile for bin-notifier-api"
```

---

## Task 14: CI/release wiring for the API

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Inspect the existing CI workflow**

Run: `cat .github/workflows/ci.yml`
Identify the build matrix (or single build step) for `cmd/notifier`.

- [ ] **Step 2: Add `cmd/api` to CI**

Edit `.github/workflows/ci.yml`. Add a build step (or matrix entry) that runs `go build ./cmd/api`. Keep `go test ./...` (it already covers the new packages).

- [ ] **Step 3: Inspect the existing release workflow**

Run: `cat .github/workflows/release.yml`
Identify the binary matrix (linux/amd64, linux/arm64, darwin/arm64) and the Docker build/push step.

- [ ] **Step 4: Add `bin-notifier-api` binary + Docker image to releases**

Edit `.github/workflows/release.yml`. For each platform in the existing notifier matrix, add an analogous build of `./cmd/api` producing `bin-notifier-api-<os>-<arch>`. Add a Docker buildx step using `Dockerfile.api` that pushes `ghcr.io/<owner>/bin-notifier-api:<tag>` (mirror the metadata-action setup used for the notifier image).

- [ ] **Step 5: Lint the workflows**

Run: `go test ./... && go build ./...`
Expected: PASS (workflow YAML lint happens in CI; locally just confirm code still builds).

- [ ] **Step 6: Commit**

```bash
git add .github/workflows
git commit -m "ci: build and release bin-notifier-api"
```

---

## Task 15: Python MCP project scaffold

**Files:**
- Create: `mcp/pyproject.toml`
- Create: `mcp/src/bin_notifier_mcp/__init__.py`
- Create: `mcp/src/bin_notifier_mcp/__main__.py`
- Create: `mcp/.gitignore`

- [ ] **Step 1: Verify uv is available**

Run: `uv --version`
Expected: prints a version. If not installed: `curl -LsSf https://astral.sh/uv/install.sh | sh`.

- [ ] **Step 2: Create the project skeleton**

Create `mcp/pyproject.toml`:

```toml
[project]
name = "bin-notifier-mcp"
version = "0.1.0"
description = "MCP server for bin-notifier"
requires-python = ">=3.12"
dependencies = [
    "mcp[cli]>=1.2.0",
    "httpx>=0.27.0",
]

[project.scripts]
bin-notifier-mcp = "bin_notifier_mcp.__main__:main"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/bin_notifier_mcp"]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.23",
    "respx>=0.21",
]
```

Create `mcp/src/bin_notifier_mcp/__init__.py`:

```python
__all__ = ["__version__"]
__version__ = "0.1.0"
```

Create `mcp/src/bin_notifier_mcp/__main__.py`:

```python
def main() -> None:
    from .server import run
    run()


if __name__ == "__main__":
    main()
```

Create `mcp/.gitignore`:

```
.venv/
__pycache__/
*.pyc
.pytest_cache/
dist/
*.egg-info/
```

- [ ] **Step 3: Initialise the lock file and install deps**

Run: `cd mcp && uv sync`
Expected: creates `.venv` and `uv.lock`.

- [ ] **Step 4: Verify the package is importable**

Run: `cd mcp && uv run python -c "import bin_notifier_mcp; print(bin_notifier_mcp.__version__)"`
Expected: `0.1.0`.

- [ ] **Step 5: Commit**

```bash
git add mcp/pyproject.toml mcp/uv.lock mcp/.gitignore mcp/src
git commit -m "feat(mcp): python project scaffold"
```

---

## Task 16: Python API client

**Files:**
- Create: `mcp/src/bin_notifier_mcp/client.py`
- Create: `mcp/tests/__init__.py`
- Create: `mcp/tests/test_client.py`

- [ ] **Step 1: Write failing tests**

Create `mcp/tests/__init__.py` (empty).

Create `mcp/tests/test_client.py`:

```python
import pytest
import httpx
import respx

from bin_notifier_mcp.client import BinNotifierClient, ApiError, NotFound, NoData


@pytest.mark.asyncio
async def test_list_locations_returns_parsed_json():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations").mock(
            return_value=httpx.Response(200, json=[{"label": "Home", "postcode": "RG12"}])
        )
        client = BinNotifierClient("https://api.example", "tok")
        got = await client.list_locations()
        assert got == [{"label": "Home", "postcode": "RG12"}]


@pytest.mark.asyncio
async def test_get_next_collection_passes_type_filter_and_token():
    async with respx.mock(base_url="https://api.example") as mock:
        route = mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(200, json={
                "location": "Home", "scraped_at": "2026-05-05T18:00:00Z",
                "date": "2026-05-08", "bin_types": ["Food Waste"],
            })
        )
        client = BinNotifierClient("https://api.example", "tok")
        got = await client.get_next_collection("Home", bin_type="Food Waste")
        assert got["date"] == "2026-05-08"
        called = route.calls.last
        assert called.request.url.params["type"] == "Food Waste"
        assert called.request.headers["authorization"] == "Bearer tok"


@pytest.mark.asyncio
async def test_404_no_match_raises_not_found():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(404, json={"error": "no matching collection", "code": "no_match"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(NotFound):
            await client.get_next_collection("Home")


@pytest.mark.asyncio
async def test_503_raises_no_data():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(503, json={"error": "no data", "code": "no_data"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(NoData):
            await client.get_next_collection("Home")


@pytest.mark.asyncio
async def test_other_errors_raise_api_error():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations").mock(return_value=httpx.Response(500, text="boom"))
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(ApiError):
            await client.list_locations()
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mcp && uv run pytest -q`
Expected: FAIL — `client` module does not exist.

- [ ] **Step 3: Implement the client**

Create `mcp/src/bin_notifier_mcp/client.py`:

```python
from __future__ import annotations

from typing import Any

import httpx


class ApiError(RuntimeError):
    """Generic API error."""


class NotFound(ApiError):
    """404 from the API (unknown location or no matching collection)."""


class NoData(ApiError):
    """503 from the API (location known but no cache yet)."""


class Unauthorized(ApiError):
    """401 from the API."""


class BinNotifierClient:
    def __init__(self, base_url: str, token: str, *, timeout: float = 5.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._token = token
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    async def _get(self, path: str, params: dict[str, Any] | None = None) -> Any:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            try:
                resp = await c.get(self._base_url + path, params=params, headers=self._headers())
            except httpx.HTTPError as e:
                raise ApiError(f"network error contacting bin-notifier API: {e}") from e

        if resp.status_code == 401:
            raise Unauthorized("unauthorized")
        if resp.status_code == 404:
            raise NotFound(_msg(resp))
        if resp.status_code == 503:
            raise NoData(_msg(resp))
        if resp.status_code >= 400:
            raise ApiError(f"API returned {resp.status_code}: {resp.text}")
        return resp.json()

    async def list_locations(self) -> list[dict[str, str]]:
        return await self._get("/v1/locations")

    async def list_collections(
        self, label: str, *, from_date: str | None = None, bin_types: list[str] | None = None
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if from_date:
            params["from"] = from_date
        if bin_types:
            params["type"] = bin_types
        return await self._get(f"/v1/locations/{label}/collections", params=params or None)

    async def get_next_collection(
        self, label: str, *, bin_type: str | None = None, from_date: str | None = None
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if from_date:
            params["from"] = from_date
        if bin_type:
            params["type"] = bin_type
        return await self._get(f"/v1/locations/{label}/collections/next", params=params or None)


def _msg(resp: httpx.Response) -> str:
    try:
        return str(resp.json().get("error") or resp.text)
    except ValueError:
        return resp.text
```

- [ ] **Step 4: Configure pytest for asyncio**

Append to `mcp/pyproject.toml`:

```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
```

- [ ] **Step 5: Run tests**

Run: `cd mcp && uv run pytest -q`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mcp/src/bin_notifier_mcp/client.py mcp/tests mcp/pyproject.toml
git commit -m "feat(mcp): async API client with typed errors"
```

---

## Task 17: Python MCP server — tools

**Files:**
- Create: `mcp/src/bin_notifier_mcp/server.py`
- Create: `mcp/tests/test_server.py`

- [ ] **Step 1: Write failing tests for tool behaviour**

Create `mcp/tests/test_server.py`:

```python
import pytest

from bin_notifier_mcp import server


class FakeClient:
    def __init__(self):
        self.locations = [{"label": "Home", "postcode": "RG12"}]
        self.next_response = {
            "location": "Home", "scraped_at": "2026-05-05T18:00:00Z",
            "date": "2026-05-07", "bin_types": ["General Waste"],
        }

    async def list_locations(self):
        return self.locations

    async def get_next_collection(self, label, *, bin_type=None, from_date=None):
        if bin_type == "Garden":
            from bin_notifier_mcp.client import NotFound
            raise NotFound("no matching collection")
        return self.next_response


@pytest.fixture
def env(monkeypatch, tmp_path):
    monkeypatch.setenv("BN_API_BASE_URL", "https://api.example")
    monkeypatch.setenv("BN_API_TOKEN", "tok")
    monkeypatch.setenv("BN_DEFAULT_LOCATION", "Home")
    return None


@pytest.mark.asyncio
async def test_list_locations_tool(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.list_locations_tool()
    assert out == [{"label": "Home", "postcode": "RG12"}]


@pytest.mark.asyncio
async def test_get_next_collection_uses_default_location(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_tool()
    assert out["location"] == "Home"
    assert out["date"] == "2026-05-07"
    assert out["bin_types"] == ["General Waste"]
    assert "days_until" in out


@pytest.mark.asyncio
async def test_get_next_collection_of_type_returns_message_when_missing(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_of_type_tool("Garden")
    assert "no" in out["message"].lower()


@pytest.mark.asyncio
async def test_missing_default_location_with_multiple_locations_errors(monkeypatch):
    monkeypatch.setenv("BN_API_BASE_URL", "https://api.example")
    monkeypatch.setenv("BN_API_TOKEN", "tok")
    monkeypatch.delenv("BN_DEFAULT_LOCATION", raising=False)

    fake = FakeClient()
    fake.locations = [{"label": "Home", "postcode": "RG12"}, {"label": "Office", "postcode": "RG40"}]
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_tool()
    assert "location" in out["error"].lower()
    assert "Home" in out["error"] and "Office" in out["error"]
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mcp && uv run pytest -q`
Expected: FAIL — `server` module does not exist.

- [ ] **Step 3: Implement the server**

Create `mcp/src/bin_notifier_mcp/server.py`:

```python
from __future__ import annotations

import os
from datetime import date
from typing import Any

from mcp.server.fastmcp import FastMCP

from .client import BinNotifierClient, NotFound, NoData

mcp = FastMCP("bin-notifier")


def _make_client() -> BinNotifierClient:
    base = os.environ.get("BN_API_BASE_URL")
    token = os.environ.get("BN_API_TOKEN")
    if not base or not token:
        raise RuntimeError("BN_API_BASE_URL and BN_API_TOKEN must be set")
    return BinNotifierClient(base, token)


async def _resolve_location(client: BinNotifierClient, label: str | None) -> str | dict[str, str]:
    if label:
        return label
    default = os.environ.get("BN_DEFAULT_LOCATION")
    if default:
        return default
    locs = await client.list_locations()
    if len(locs) == 1:
        return locs[0]["label"]
    labels = ", ".join(l["label"] for l in locs)
    return {"error": f"location is required (configured: {labels})"}


def _days_until(target: str) -> int:
    return (date.fromisoformat(target) - date.today()).days


@mcp.tool(name="list_locations", description="List configured bin-notifier locations.")
async def list_locations_tool() -> list[dict[str, str]]:
    return await _make_client().list_locations()


@mcp.tool(
    name="get_next_collection",
    description="Return the next bin collection day for a location. Omit `location` to use the default.",
)
async def get_next_collection_tool(location: str | None = None) -> dict[str, Any]:
    client = _make_client()
    label = await _resolve_location(client, location)
    if isinstance(label, dict):
        return label
    try:
        resp = await client.get_next_collection(label)
    except NoData:
        return {"error": f"no data cached yet for {label}"}
    except NotFound:
        return {"error": f"no upcoming collection found for {label}"}
    resp["days_until"] = _days_until(resp["date"])
    return resp


@mcp.tool(
    name="get_next_collection_of_type",
    description="Return the next collection of a specific bin type (e.g. 'Food Waste').",
)
async def get_next_collection_of_type_tool(bin_type: str, location: str | None = None) -> dict[str, Any]:
    client = _make_client()
    label = await _resolve_location(client, location)
    if isinstance(label, dict):
        return label
    try:
        resp = await client.get_next_collection(label, bin_type=bin_type)
    except NoData:
        return {"error": f"no data cached yet for {label}"}
    except NotFound:
        return {"message": f"no upcoming {bin_type} collection found for {label}"}
    resp["days_until"] = _days_until(resp["date"])
    return resp


def run() -> None:
    mcp.run()
```

- [ ] **Step 4: Run tests**

Run: `cd mcp && uv run pytest -q`
Expected: PASS.

- [ ] **Step 5: Smoke-test the entry point**

Run: `cd mcp && BN_API_BASE_URL=x BN_API_TOKEN=y uv run bin-notifier-mcp --help 2>&1 | head -5 || true`
Expected: starts without ImportError (it will hang on stdio in real use; Ctrl-C to exit).

- [ ] **Step 6: Commit**

```bash
git add mcp/src/bin_notifier_mcp/server.py mcp/tests/test_server.py
git commit -m "feat(mcp): FastMCP server with three tools"
```

---

## Task 18: Dockerfile for the Python MCP

**Files:**
- Create: `mcp/Dockerfile`

- [ ] **Step 1: Write the Dockerfile**

Create `mcp/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1.7
FROM python:3.12-slim AS build
WORKDIR /app
ENV UV_LINK_MODE=copy
COPY --from=ghcr.io/astral-sh/uv:0.5.0 /uv /usr/local/bin/uv
COPY pyproject.toml uv.lock ./
COPY src ./src
RUN uv sync --frozen --no-dev
RUN uv build --wheel

FROM python:3.12-slim
WORKDIR /app
COPY --from=build /app/dist/*.whl /tmp/
RUN pip install --no-cache-dir /tmp/*.whl && rm /tmp/*.whl
USER 1000:1000
ENTRYPOINT ["bin-notifier-mcp"]
```

- [ ] **Step 2: Build the image**

Run: `cd mcp && docker build -t bin-notifier-mcp:test .`
Expected: builds successfully.

- [ ] **Step 3: Smoke test the image**

Run:
```bash
docker run --rm bin-notifier-mcp:test --help 2>&1 | head -5 || true
```
Expected: process starts (will exit because stdio isn't connected — exit code is OK).

- [ ] **Step 4: Commit**

```bash
git add mcp/Dockerfile
git commit -m "build(mcp): Dockerfile for Python MCP server"
```

---

## Task 19: Helm chart — base files

**Files:**
- Create: `deploy/helm/bin-notifier/Chart.yaml`
- Create: `deploy/helm/bin-notifier/.helmignore`
- Create: `deploy/helm/bin-notifier/values.yaml`
- Create: `deploy/helm/bin-notifier/templates/_helpers.tpl`
- Create: `deploy/helm/bin-notifier/templates/NOTES.txt`

- [ ] **Step 1: Confirm Helm is available**

Run: `helm version --short`
Expected: prints a v3.x version. Install via `brew install helm` if missing.

- [ ] **Step 2: Create `Chart.yaml`**

```yaml
apiVersion: v2
name: bin-notifier
description: bin-notifier API + nightly scrape CronJob
type: application
version: 0.1.0
appVersion: "0.1.0"
```

- [ ] **Step 3: Create `.helmignore`**

```
.git
.DS_Store
*.tgz
```

- [ ] **Step 4: Create `values.yaml`**

```yaml
image:
  api:
    repository: ghcr.io/stebennett/bin-notifier-api
    tag: ""           # defaults to .Chart.AppVersion
    pullPolicy: IfNotPresent
  notifier:
    repository: ghcr.io/stebennett/bin-notifier
    tag: ""
    pullPolicy: IfNotPresent

api:
  replicaCount: 1     # SQLite single-writer; the only safe value.
  service:
    port: 80
  ingress:
    enabled: false
    className: ""
    host: ""
    tls:
      enabled: false
      secretName: ""
    annotations: {}
  persistence:
    size: 1Gi
    storageClass: local-path
  resources: {}

notifier:
  schedule: "0 18 * * *"
  dryRun: false
  resources: {}
  # Twilio creds: chart never templates these. Reference an existing Secret.
  twilio:
    existingSecret: ""    # required: must define TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN

# Configuration source: set `config:` to inline YAML or `existingConfigSecret:` to reuse one.
config: ""              # YAML string (config.yaml contents)
existingConfigSecret: ""

# Tokens. If both unset and existingTokensSecret is empty, the chart auto-generates them on install.
# (Helm cannot read existing values during upgrade — set explicit tokens or use existingTokensSecret
# to avoid losing the auto-generated values across `helm upgrade --reset-values`.)
tokens:
  readToken: ""
  writeToken: ""
existingTokensSecret: ""    # must define keys: read-token, write-token
```

- [ ] **Step 5: Create `templates/_helpers.tpl`**

```yaml
{{- define "bin-notifier.fullname" -}}
{{- printf "%s" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "bin-notifier.labels" -}}
app.kubernetes.io/name: bin-notifier
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "bin-notifier.apiImage" -}}
{{- $tag := .Values.image.api.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.api.repository $tag -}}
{{- end -}}

{{- define "bin-notifier.notifierImage" -}}
{{- $tag := .Values.image.notifier.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.notifier.repository $tag -}}
{{- end -}}

{{- define "bin-notifier.configSecretName" -}}
{{- if .Values.existingConfigSecret -}}
{{ .Values.existingConfigSecret }}
{{- else -}}
{{ include "bin-notifier.fullname" . }}-config
{{- end -}}
{{- end -}}

{{- define "bin-notifier.tokensSecretName" -}}
{{- if .Values.existingTokensSecret -}}
{{ .Values.existingTokensSecret }}
{{- else -}}
{{ include "bin-notifier.fullname" . }}-tokens
{{- end -}}
{{- end -}}
```

- [ ] **Step 6: Create `templates/NOTES.txt`**

```
bin-notifier installed.

API service: {{ include "bin-notifier.fullname" . }}-api (port {{ .Values.api.service.port }})
{{- if .Values.api.ingress.enabled }}
Ingress: https://{{ .Values.api.ingress.host }}/healthz
{{- end }}

Tokens secret: {{ include "bin-notifier.tokensSecretName" . }}
  (keys: read-token, write-token)

Nightly schedule: {{ .Values.notifier.schedule }}
```

- [ ] **Step 7: Lint the partial chart**

Run: `helm lint deploy/helm/bin-notifier --set notifier.twilio.existingSecret=fake-twilio --set config="from_number: '+1'\nto_number: '+2'\nlocations: [{label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}]"`
Expected: PASS (warnings about missing templates are fine — we add them next).

- [ ] **Step 8: Commit**

```bash
git add deploy/helm/bin-notifier
git commit -m "feat(helm): chart skeleton with values and helpers"
```

---

## Task 20: Helm chart — Secrets templates

**Files:**
- Create: `deploy/helm/bin-notifier/templates/secret-config.yaml`
- Create: `deploy/helm/bin-notifier/templates/secret-tokens.yaml`

- [ ] **Step 1: Create `secret-config.yaml`**

```yaml
{{- if not .Values.existingConfigSecret }}
{{- if not .Values.config }}{{- fail "set values.config (inline YAML) or values.existingConfigSecret" }}{{- end }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "bin-notifier.configSecretName" . }}
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
type: Opaque
stringData:
  config.yaml: |
{{ .Values.config | indent 4 }}
{{- end }}
```

- [ ] **Step 2: Create `secret-tokens.yaml`**

```yaml
{{- if not .Values.existingTokensSecret }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "bin-notifier.tokensSecretName" . }}
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
  annotations:
    "helm.sh/resource-policy": keep
type: Opaque
stringData:
  read-token: {{ .Values.tokens.readToken | default (randAlphaNum 40) | quote }}
  write-token: {{ .Values.tokens.writeToken | default (randAlphaNum 40) | quote }}
{{- end }}
```

> Note: `helm.sh/resource-policy: keep` plus pinning explicit token values is the recommended way to avoid token churn on upgrade. The chart's NOTES should remind users to capture and re-supply the tokens (or use `existingTokensSecret`) for any future upgrade.

- [ ] **Step 3: Render and inspect**

Run:
```bash
helm template t deploy/helm/bin-notifier \
  --set notifier.twilio.existingSecret=fake-twilio \
  --set-string config="from_number: '+1'
to_number: '+2'
locations:
  - label: Home
    scraper: bracknell
    postcode: P
    address_code: A
    collection_days:
      - day: monday
        types: [General Waste]
"
```
Expected: prints two `kind: Secret` documents with sane values.

- [ ] **Step 4: Commit**

```bash
git add deploy/helm/bin-notifier/templates/secret-config.yaml deploy/helm/bin-notifier/templates/secret-tokens.yaml
git commit -m "feat(helm): config and tokens Secret templates"
```

---

## Task 21: Helm chart — API Deployment, Service, PVC, Ingress

**Files:**
- Create: `deploy/helm/bin-notifier/templates/api-pvc.yaml`
- Create: `deploy/helm/bin-notifier/templates/api-deployment.yaml`
- Create: `deploy/helm/bin-notifier/templates/api-service.yaml`
- Create: `deploy/helm/bin-notifier/templates/api-ingress.yaml`

- [ ] **Step 1: Create `api-pvc.yaml`**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "bin-notifier.fullname" . }}-api
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: {{ .Values.api.persistence.size }}
  {{- if .Values.api.persistence.storageClass }}
  storageClassName: {{ .Values.api.persistence.storageClass }}
  {{- end }}
```

- [ ] **Step 2: Create `api-deployment.yaml`**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "bin-notifier.fullname" . }}-api
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.api.replicaCount }}
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app.kubernetes.io/name: bin-notifier
      app.kubernetes.io/instance: {{ .Release.Name }}
      app.kubernetes.io/component: api
  template:
    metadata:
      labels:
        app.kubernetes.io/name: bin-notifier
        app.kubernetes.io/instance: {{ .Release.Name }}
        app.kubernetes.io/component: api
    spec:
      containers:
        - name: api
          image: {{ include "bin-notifier.apiImage" . }}
          imagePullPolicy: {{ .Values.image.api.pullPolicy }}
          args: ["-c", "/etc/bin-notifier/config.yaml"]
          env:
            - name: BN_API_DB_PATH
              value: /var/lib/bin-notifier/cache.db
            - name: BN_API_LISTEN_ADDR
              value: ":8080"
            - name: BN_API_READ_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "bin-notifier.tokensSecretName" . }}
                  key: read-token
            - name: BN_API_WRITE_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "bin-notifier.tokensSecretName" . }}
                  key: write-token
          ports:
            - containerPort: 8080
              name: http
          readinessProbe:
            httpGet: { path: /healthz, port: http }
          livenessProbe:
            httpGet: { path: /healthz, port: http }
            initialDelaySeconds: 10
          volumeMounts:
            - name: config
              mountPath: /etc/bin-notifier
              readOnly: true
            - name: data
              mountPath: /var/lib/bin-notifier
          resources: {{- toYaml .Values.api.resources | nindent 12 }}
      volumes:
        - name: config
          secret:
            secretName: {{ include "bin-notifier.configSecretName" . }}
            items:
              - key: config.yaml
                path: config.yaml
        - name: data
          persistentVolumeClaim:
            claimName: {{ include "bin-notifier.fullname" . }}-api
```

- [ ] **Step 3: Create `api-service.yaml`**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "bin-notifier.fullname" . }}-api
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
spec:
  type: ClusterIP
  ports:
    - port: {{ .Values.api.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: bin-notifier
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/component: api
```

- [ ] **Step 4: Create `api-ingress.yaml`**

```yaml
{{- if .Values.api.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "bin-notifier.fullname" . }}-api
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
  {{- with .Values.api.ingress.annotations }}
  annotations: {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.api.ingress.className }}
  ingressClassName: {{ .Values.api.ingress.className }}
  {{- end }}
  {{- if .Values.api.ingress.tls.enabled }}
  tls:
    - hosts: [{{ .Values.api.ingress.host | quote }}]
      secretName: {{ .Values.api.ingress.tls.secretName }}
  {{- end }}
  rules:
    - host: {{ .Values.api.ingress.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ include "bin-notifier.fullname" . }}-api
                port:
                  number: {{ .Values.api.service.port }}
{{- end }}
```

- [ ] **Step 5: Render and inspect**

Run:
```bash
helm template t deploy/helm/bin-notifier \
  --set notifier.twilio.existingSecret=fake-twilio \
  --set api.ingress.enabled=true --set api.ingress.host=bn.example \
  --set-string config="from_number: '+1'
to_number: '+2'
locations:
  - {label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}
" | grep -E '^kind:'
```
Expected: includes `Deployment`, `Service`, `Ingress`, `PersistentVolumeClaim`.

- [ ] **Step 6: Commit**

```bash
git add deploy/helm/bin-notifier/templates/api-deployment.yaml deploy/helm/bin-notifier/templates/api-service.yaml deploy/helm/bin-notifier/templates/api-ingress.yaml deploy/helm/bin-notifier/templates/api-pvc.yaml
git commit -m "feat(helm): API Deployment, Service, PVC, Ingress"
```

---

## Task 22: Helm chart — notifier CronJob

**Files:**
- Create: `deploy/helm/bin-notifier/templates/notifier-cronjob.yaml`

- [ ] **Step 1: Create the CronJob template**

```yaml
{{- if not .Values.notifier.twilio.existingSecret }}
{{- fail "values.notifier.twilio.existingSecret is required (must define TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN)" }}
{{- end }}
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{ include "bin-notifier.fullname" . }}-notifier
  labels: {{- include "bin-notifier.labels" . | nindent 4 }}
spec:
  schedule: {{ .Values.notifier.schedule | quote }}
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: notifier
              image: {{ include "bin-notifier.notifierImage" . }}
              imagePullPolicy: {{ .Values.image.notifier.pullPolicy }}
              args:
                - "-c"
                - "/etc/bin-notifier/config.yaml"
                {{- if .Values.notifier.dryRun }}
                - "-x"
                {{- end }}
              env:
                - name: BN_API_BASE_URL
                  value: "http://{{ include "bin-notifier.fullname" . }}-api:{{ .Values.api.service.port }}"
                - name: BN_API_WRITE_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: {{ include "bin-notifier.tokensSecretName" . }}
                      key: write-token
                - name: TWILIO_ACCOUNT_SID
                  valueFrom:
                    secretKeyRef:
                      name: {{ .Values.notifier.twilio.existingSecret }}
                      key: TWILIO_ACCOUNT_SID
                - name: TWILIO_AUTH_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: {{ .Values.notifier.twilio.existingSecret }}
                      key: TWILIO_AUTH_TOKEN
              volumeMounts:
                - name: config
                  mountPath: /etc/bin-notifier
                  readOnly: true
              resources: {{- toYaml .Values.notifier.resources | nindent 16 }}
          volumes:
            - name: config
              secret:
                secretName: {{ include "bin-notifier.configSecretName" . }}
                items:
                  - key: config.yaml
                    path: config.yaml
```

- [ ] **Step 2: Lint and render**

Run:
```bash
helm lint deploy/helm/bin-notifier \
  --set notifier.twilio.existingSecret=fake-twilio \
  --set-string config="from_number: '+1'
to_number: '+2'
locations:
  - {label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}
"
```
Expected: PASS.

Run the same with `helm template` and confirm a `CronJob` document is rendered with the expected env vars.

- [ ] **Step 3: Verify the failure path**

Run: `helm template t deploy/helm/bin-notifier --set-string config="x: y"`
Expected: FAIL with the "twilio.existingSecret is required" message.

- [ ] **Step 4: Commit**

```bash
git add deploy/helm/bin-notifier/templates/notifier-cronjob.yaml
git commit -m "feat(helm): notifier CronJob template"
```

---

## Task 23: Helm chart — CI lint job

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add a Helm lint step**

Edit `.github/workflows/ci.yml`. Add (or extend) a job that runs:

```yaml
      - name: Set up Helm
        uses: azure/setup-helm@v4
        with:
          version: v3.14.0
      - name: Helm lint
        run: |
          helm lint deploy/helm/bin-notifier \
            --set notifier.twilio.existingSecret=fake-twilio \
            --set-string config="from_number: '+1'
          to_number: '+2'
          locations:
            - {label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}
          "
      - name: Helm template
        run: |
          helm template ci deploy/helm/bin-notifier \
            --set notifier.twilio.existingSecret=fake-twilio \
            --set-string config="from_number: '+1'
          to_number: '+2'
          locations:
            - {label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}
          " > /tmp/rendered.yaml
          test -s /tmp/rendered.yaml
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: lint and template the Helm chart"
```

---

## Task 24: README, CLAUDE.md, and config docs

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Create: `mcp/README.md`

- [ ] **Step 1: Update `CLAUDE.md`**

Add a new section under "Build and Run Commands":

```markdown
# Build the API server
go build -o bin-notifier-api ./cmd/api

# Run the API server
BN_API_CONFIG_FILE=config.yaml \
BN_API_DB_PATH=/tmp/cache.db \
BN_API_READ_TOKEN=$(openssl rand -hex 16) \
BN_API_WRITE_TOKEN=$(openssl rand -hex 16) \
go run ./cmd/api

# Build the Python MCP server (uv-managed)
cd mcp && uv sync

# Run the Python MCP server (stdio)
cd mcp && BN_API_BASE_URL=http://localhost:8080 BN_API_TOKEN=<read-token> uv run bin-notifier-mcp
```

Add to "Project Structure":

```
cmd/api/             - bin-notifier-api: HTTP server fronting the SQLite cache
pkg/api/             - HTTP handlers, server bootstrap, auth middleware
pkg/store/           - SQLite-backed cache (modernc.org/sqlite, pure Go)
pkg/apiclient/       - Go HTTP client used by the notifier to push to the API

mcp/                 - Python/uv FastMCP server
deploy/helm/bin-notifier/  - Helm chart (API + notifier CronJob)
```

Add to "Environment Variables":

```
**API server:**
- `BN_API_CONFIG_FILE` - YAML config path (or use -c)
- `BN_API_DB_PATH` - SQLite file path (default /var/lib/bin-notifier/cache.db)
- `BN_API_LISTEN_ADDR` - listen address (default :8080)
- `BN_API_READ_TOKEN` / `BN_API_WRITE_TOKEN` - bearer tokens

**Notifier (new):**
- `BN_API_BASE_URL` - if set, notifier PUTs scraped data here after each scrape
- `BN_API_WRITE_TOKEN` - bearer token for the API's write endpoint
```

- [ ] **Step 2: Update `README.md`**

Add a high-level architecture summary near the top describing the three-component split (notifier CronJob, API, Python MCP) and link to the Helm chart and `mcp/README.md`.

- [ ] **Step 3: Create `mcp/README.md`**

```markdown
# bin-notifier MCP server

Python FastMCP server (stdio transport) that exposes bin collection data from a running `bin-notifier-api`.

## Tools
- `list_locations` — list configured locations.
- `get_next_collection(location?)` — next collection day for a location.
- `get_next_collection_of_type(bin_type, location?)` — next collection of a specific type.

## Configuration
Environment variables:
- `BN_API_BASE_URL` — e.g. `http://bin-notifier-api.bin-notifier.svc:80` or `https://bn.example.com`
- `BN_API_TOKEN` — read token issued by the API
- `BN_DEFAULT_LOCATION` — optional default location label

## Running locally
```bash
cd mcp
uv sync
BN_API_BASE_URL=http://localhost:8080 BN_API_TOKEN=... uv run bin-notifier-mcp
```

## Docker
```bash
docker build -t bin-notifier-mcp .
docker run --rm -i \
  -e BN_API_BASE_URL=https://bn.example.com \
  -e BN_API_TOKEN=... \
  bin-notifier-mcp
```
```

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md README.md mcp/README.md
git commit -m "docs: document API server, Python MCP, and Helm chart"
```

---

## Task 25: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 2: Run all Go tests**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 3: Run Python tests**

Run: `cd mcp && uv run pytest -q`
Expected: all pass.

- [ ] **Step 4: Lint Helm**

Run:
```bash
helm lint deploy/helm/bin-notifier \
  --set notifier.twilio.existingSecret=fake-twilio \
  --set-string config="from_number: '+1'
to_number: '+2'
locations:
  - {label: Home, scraper: bracknell, postcode: P, address_code: A, collection_days: [{day: monday, types: [General Waste]}]}
"
```
Expected: PASS.

- [ ] **Step 5: End-to-end smoke (manual)**

Run the API in a terminal, then in another:
```bash
TOK=...; curl -s -H "Authorization: Bearer $WRITE_TOK" -H "Content-Type: application/json" \
  -X PUT http://localhost:8080/v1/locations/Home/collections \
  -d '{"scraped_at":"2026-05-05T18:00:00Z","collections":[{"bin_type":"General Waste","date":"2026-05-07"}]}'
curl -s -H "Authorization: Bearer $READ_TOK" http://localhost:8080/v1/locations/Home/collections/next | jq .
```
Expected: PUT returns 204; GET returns the seeded next collection.

- [ ] **Step 6: Confirm the spec is fully implemented**

Re-read `docs/superpowers/specs/2026-05-05-bin-notifier-api-and-python-mcp-design.md` and check each section against the implemented code. Note any gaps in the PR description.
