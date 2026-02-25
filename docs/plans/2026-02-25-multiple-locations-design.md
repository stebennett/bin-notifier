# Multiple Locations Support — Design

## Context

bin-notifier currently supports a single location per run. This refactor adds support for 2-3 locations in a single invocation, each potentially using a different council website scraper.

## Requirements

- Multiple properties, all notifying one recipient
- YAML config file for location management (replaces CLI flags for location config)
- One SMS per location with a required human-readable label
- User specifies which scraper to use per location
- Partial failure OK: continue for locations that succeed, exit non-zero if any fail
- No backward compatibility required — full migration to new config approach
- Human-readable day names for collection day (e.g. "tuesday" not "2")

## Configuration

### Config file (`config.yaml`)

```yaml
from_number: "+441234567890"
to_number: "+449876543210"

locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday
  - label: Office
    scraper: wokingham
    postcode: "RG42 2XY"
    address_code: "67890"
    collection_day: thursday
```

### CLI flags

Only three flags remain:

- `-c / --config` — path to config file (required). Env var: `BN_CONFIG_FILE`
- `-x / --dryrun` — dry-run mode. Env var: `BN_DRY_RUN`
- `-d / --todaydate` — date override for testing. Env var: `BN_TODAY_DATE`

Twilio credentials remain as environment variables (`TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`).

### Validation

- Unknown scraper names rejected
- Missing required fields rejected (label, scraper, postcode, address_code, collection_day)
- Invalid day names rejected (must be one of monday-sunday, case-insensitive)
- At least one location required

## Scraper Architecture

### Interface

```go
type BinScraper interface {
    BinTimes(postcode, addressCode string) ([]BinTime, error)
}
```

### Package structure

```
pkg/scraper/
├── scraper.go          # BinScraper interface, BinTime type, registry
├── bracknell.go        # Bracknell Forest scraper (refactored from current code)
├── bracknell_test.go
├── wokingham.go        # Wokingham scraper (stubbed until URL provided)
└── wokingham_test.go
```

### Registry

A `NewScraper(name string) (BinScraper, error)` function returns the appropriate scraper or an error for unknown names.

## Notifier Orchestration

### Flow

1. Parse config file, validate all locations
2. For each location:
   - Look up the scraper by name
   - Scrape bin times for that location's postcode/address code
   - Check if any collections are due tomorrow
   - If collections found, send SMS with location label prefix
   - If tomorrow is regular collection day but no collections, send SMS noting that
   - If scraping fails, log the error and continue to next location
3. Exit with non-zero status if any location had errors

### SMS message format

```
Home: Recycling, Food collection tomorrow
```

```
Home: No collections scheduled for tomorrow (regular collection day)
```

### Return type

`Run()` returns `[]NotificationResult` (one per location) plus an aggregate error.

## Testing Strategy

### Config parsing

- Valid YAML with multiple locations
- Human-readable day names (case-insensitive)
- Invalid day names rejected
- Missing required fields rejected
- Unknown scraper names rejected

### Scraper registry

- Known names return correct implementation
- Unknown names return error

### Notifier orchestration

- Multiple locations with collections tomorrow: one SMS per location
- One location fails, other succeeds: SMS for successful, non-zero exit
- No collections at any location: no SMS
- Location label appears in SMS text

### Unchanged

Existing tests for Bracknell scraper parsing, Twilio client, dateutil, regexp remain as-is.

## Changes Summary

| Area | Change |
|---|---|
| `pkg/config/` | YAML config file parsing. `Config` holds `[]Location` + shared fields. CLI reduced to `-c`, `-x`, `-d`. |
| `pkg/scraper/` | Extract interface + `BinTime`. Refactor Bracknell into own file. Add registry. Stub Wokingham. |
| `cmd/notifier/` | `Run()` loops over locations, SMS per location with label, partial failure support. |
| `pkg/clients/` | No changes. |
| `pkg/dateutil/` | Add day name parsing. |
| `Dockerfile` | Update entrypoint for new CLI flags. |
| `README.md` | Update usage docs. |

## Out of Scope

- Web admin UI
- Wokingham scraper implementation (stubbed until URL provided)
- Notification channels other than Twilio SMS
