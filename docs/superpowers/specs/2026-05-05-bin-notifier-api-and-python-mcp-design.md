# bin-notifier HTTP API and Python MCP Server — Design

**Date:** 2026-05-05
**Status:** Approved

## Goal

Replace the existing Go MCP server with a Python/uv MCP server that consumes a new HTTP API exposed by bin-notifier. The nightly scrape continues to send SMS (existing behavior) and additionally pushes scraped collection data to the API, which caches it in SQLite for the MCP server to read.

## Non-Goals

- No on-demand scraping triggered by MCP queries — all scraping is nightly.
- No multi-user or multi-tenant support — single-user system.
- No projection from config rules (`pkg/schedule`) — cache holds scraped data only.
- No horizontal scaling of the API — single replica is the only supported topology.

## Architecture

Three deployable artifacts, all in this repo:

1. **`bin-notifier-api`** (new Go binary, `cmd/api/`) — long-running HTTP server. Owns the SQLite cache file (mounted on its pod only). Exposes a versioned read API (read-token bearer auth) and a write endpoint (write-token bearer auth). Loads `config.yaml` to know about configured locations.
2. **`bin-notifier`** (existing, `cmd/notifier/`) — gains one new responsibility: after scraping each location, PUT the results to the API. Then continue with the existing SMS logic. Run as the nightly k8s `CronJob`.
3. **`bin-notifier-mcp`** (new Python project, `mcp/` directory, uv-managed) — FastMCP stdio server. Calls the API over HTTP with the read token. Deployable inside metamcp or runnable locally.

The existing Go MCP server (`cmd/server/`) is **deleted** as part of this work.

## Data Model

Single SQLite table:

```sql
CREATE TABLE collections (
    location    TEXT NOT NULL,   -- matches Location.Label from config
    bin_type    TEXT NOT NULL,   -- e.g. "General Waste", "Food Waste"
    date        TEXT NOT NULL,   -- ISO 8601 (YYYY-MM-DD)
    scraped_at  TEXT NOT NULL,   -- ISO 8601 timestamp of the scrape
    PRIMARY KEY (location, bin_type, date)
);
CREATE INDEX idx_collections_location_date ON collections(location, date);
```

**Write semantics:** the write endpoint receives a full snapshot for one location and replaces all rows for that location atomically (`DELETE WHERE location=?` + bulk `INSERT` in a transaction). Idempotent and self-healing — if the council removes a date, it disappears from the cache on the next scrape.

**SQLite mode:** WAL enabled, `modernc.org/sqlite` (pure Go, no CGO). File lives on a small PVC mounted only on the API pod.

## HTTP API

All JSON. All endpoints (read and write) use `Authorization: Bearer <token>`. The API is configured with two distinct tokens (`BN_API_READ_TOKEN`, `BN_API_WRITE_TOKEN`); middleware checks the supplied token against the role required by each route.

### Read endpoints (require read token)

- `GET /v1/locations` — list configured locations from `config.yaml`. Response: `[{label, postcode}]`.
- `GET /v1/locations/{label}/collections?from=YYYY-MM-DD&type=...` — all cached collections for a location, ordered by date ascending. `from` defaults to today; `type` is repeatable. Response: `{location, scraped_at, collections: [{bin_type, date}]}`.
- `GET /v1/locations/{label}/collections/next?type=...` — next collection day for the location, optionally filtered to a specific bin type. Response: `{location, date, bin_types: [...], scraped_at}` or 404 if nothing matches.

### Write endpoint (requires write token)

- `PUT /v1/locations/{label}/collections` — replace cached collections for a location. Body: `{scraped_at, collections: [{bin_type, date}]}`. Returns 204.

### Unauthenticated

- `GET /healthz` — returns 200 if the SQLite file is openable.

### Errors

JSON `{error: "...", code: "..."}` with appropriate HTTP statuses. 401 on missing/wrong token, 404 on unknown location or no matching collection, 503 if a known location has no cached data yet.

## Python MCP Server

**Layout:** `mcp/` directory at repo root, uv-managed (`pyproject.toml`, `uv.lock`). Single FastMCP module, stdio transport.

**Config (env vars):**
- `BN_API_BASE_URL` — e.g. `http://bin-notifier-api.bin-notifier.svc.cluster.local` in-cluster, or `https://...` externally
- `BN_API_TOKEN` — read token
- `BN_DEFAULT_LOCATION` — optional; default location label when tools omit `location`

**Tools:**

1. `get_next_collection(location: str | None = None)` — calls `GET /v1/locations/{label}/collections/next`. Returns `{date, bin_types, days_until}`.
2. `get_next_collection_of_type(bin_type: str, location: str | None = None)` — calls `GET /v1/locations/{label}/collections/next?type={bin_type}`. Returns `{date, days_until}` or a not-found message.
3. `list_locations()` — calls `GET /v1/locations`. Lets the LLM discover available labels when the user is ambiguous.

If `location` is omitted, fall back to `BN_DEFAULT_LOCATION`. If that is also unset and there is more than one configured location, return an error listing the available labels.

**HTTP client:** `httpx` with one connection-error retry, 5s timeout. No persistent state in the MCP process.

**Packaging:** `uv build` produces a wheel; a `Dockerfile` in `mcp/` builds a runnable image for metamcp. For local use, `uvx --from . bin-notifier-mcp` works directly.

## Deployment (k3s)

Helm chart at `deploy/helm/bin-notifier/` packaging the API Deployment, Service, Ingress, PVC, the notifier CronJob, and the optional Secret templates. Standard chart layout (`Chart.yaml`, `values.yaml`, `templates/`).

**Secrets in-namespace (`bin-notifier`):**
- `bin-notifier-twilio` — `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` (existing; chart never templates these — `twilio.existingSecret` is required).
- `bin-notifier-api-tokens` — `read-token`, `write-token`. Auto-generated on install via `randAlphaNum` if `tokens.readToken` / `tokens.writeToken` are unset; `existingTokensSecret` overrides.
- `bin-notifier-config` — the `config.yaml` rendered into a Secret from `values.yaml`'s `config` key, or `existingConfigSecret` for out-of-band management.

**API Deployment:**
- 1 replica (SQLite single-writer; documented as the only safe value).
- Mounts: config Secret at `/etc/bin-notifier/config.yaml`, PVC at `/var/lib/bin-notifier`.
- Env: `BN_API_READ_TOKEN`, `BN_API_WRITE_TOKEN` from the tokens Secret.
- Liveness/readiness: `GET /healthz`.
- `Recreate` strategy (not RollingUpdate) to ensure the SQLite file is never opened by two pods simultaneously.

**Service:** ClusterIP on port 80 → 8080. Optional Ingress with TLS for the local-MCP use case.

**PVC:** default 1Gi, `local-path` storage class.

**Notifier CronJob:** existing notifier binary, default schedule `0 18 * * *`. Mounts config Secret. Env adds `BN_API_BASE_URL` (in-cluster Service DNS) and `BN_API_WRITE_TOKEN`. `concurrencyPolicy: Forbid`, `successfulJobsHistoryLimit: 3`.

**Key `values.yaml` knobs:**
- `image.api.repository` / `image.api.tag` and same for `notifier`
- `api.replicaCount`, `api.persistence.size` / `storageClass`, `api.ingress.enabled` / `host` / `tls`
- `notifier.schedule`, `notifier.dryRun`
- `config` or `existingConfigSecret`
- `tokens.readToken` / `tokens.writeToken` or `existingTokensSecret`
- `twilio.existingSecret` (required)

The MCP server is **not** in the chart — it is deployed via metamcp or run locally.

## Data Flow

**Nightly:**
1. CronJob fires → notifier loads config.
2. For each location: scrape → PUT `/v1/locations/{label}/collections` to the API → existing SMS check.
3. API replaces rows for that location in a single transaction.

**On MCP query:** MCP tool → API read endpoint → SQLite query → JSON response.

## Failure Modes

- **Scrape fails for one location:** log, continue to next location, exit non-zero (existing behavior). Cache for the failed location keeps the previous day's data — stale but not wrong; `scraped_at` is exposed so the MCP/LLM can disclose staleness.
- **API push fails (network or 5xx):** notifier logs, retries once with backoff, then continues to the SMS step so SMS is not blocked by API trouble. The cache misses one update; the next nightly run heals it.
- **API down when MCP queries:** MCP surfaces a clear error message ("bin-notifier API unreachable").
- **No cache yet (fresh install):** API returns 503 with `{error: "no data cached for location X yet"}`. MCP relays it.
- **Stale cache:** API still serves the data and includes `scraped_at` in every response so the consumer can decide what to do.

## Testing

- **API:** table-driven handler tests against in-memory SQLite (`:memory:`); auth middleware tests for both token roles; integration test that spins the server on a random port and exercises a full PUT-then-GET round trip.
- **Notifier:** extend existing tests with a fake API client; verify push happens before SMS and that push failure does not block SMS.
- **MCP:** pytest with `respx` mocking `httpx`; one test per tool covering success, 404, 503, and auth-error cases.
- **Helm:** `helm lint` + `helm template` snapshot test in CI.

## Out of Scope / Future Work

- Horizontal scaling of the API (would require switching off SQLite).
- Projection of future collections from config rules when scraped data runs short.
- Multi-tenant or multi-user authorization.
- Webhook / push notifications from the API to MCP clients.
