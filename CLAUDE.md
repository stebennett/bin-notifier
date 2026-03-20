# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the application
go build -o bin-notifier ./cmd/notifier

# Run the application (requires config.yaml)
go run cmd/notifier/main.go -c config.yaml

# Run with dry-run mode (no SMS sent)
go run cmd/notifier/main.go -c config.yaml -x

# Run with a specific date (for testing date logic)
go run cmd/notifier/main.go -c config.yaml -d 2024-01-15

# Run tests
go test ./...

# Run a specific test
go test -run TestParseNextCollectionTime ./pkg/scraper/

# Build the MCP server
go build -o bin-notifier-mcp ./cmd/server

# Run the MCP server (stdio transport)
./bin-notifier-mcp -c config.yaml

# Build Docker image
docker build -t bin-notifier .

# Build MCP server Docker image
docker build -f Dockerfile.mcp -t bin-notifier-mcp .

# Run with Docker (dry-run mode)
docker run --rm \
  -e TWILIO_ACCOUNT_SID=test \
  -e TWILIO_AUTH_TOKEN=test \
  -v /path/to/config.yaml:/config.yaml:ro \
  bin-notifier -c /config.yaml -x

# Run MCP server with Docker (stdio transport, use -i for stdin)
docker run -i --rm \
  -v /path/to/config.yaml:/config.yaml:ro \
  bin-notifier-mcp -c /config.yaml
```

## Prerequisites

- Google Chrome or Chromium (required for headless web scraping)

## Environment Variables

**Twilio (required):**
- `TWILIO_ACCOUNT_SID` - Twilio account SID
- `TWILIO_AUTH_TOKEN` - Twilio auth token

**Phone numbers (optional, fallback when not set in config file):**
- `BN_FROM_NUMBER` - Twilio "from" phone number (used when `from_number` is not set in config)
- `BN_TO_NUMBER` - Destination phone number (used when `to_number` is not set in config)

**Application config (alternative to CLI flags):**
- `BN_CONFIG_FILE` - Path to YAML config file
- `BN_DRY_RUN` - Set to `true` for dry-run mode
- `BN_TODAY_DATE` - Override today's date (YYYY-MM-DD)

CLI flags take precedence over environment variables.

## Architecture

This is a Go application that scrapes bin collection schedules from council websites and sends SMS notifications via Twilio when collections are due tomorrow. It supports multiple locations with pluggable council-specific scrapers.

### Project Structure

```
cmd/notifier/
  main.go          - Entry point + Notifier struct (orchestrates the workflow)
  main_test.go     - Tests using mock scrapers and SMS client

cmd/server/
  main.go          - MCP server entry point: loads config, registers tools, runs stdio transport
  main_test.go     - Tool handler tests with mock scrapers

pkg/config/
  config.go        - Flags struct + ParseFlags() for CLI flags (-c, -x, -d)
                     Config/Location structs + LoadConfig() / LoadConfigForMCP() for YAML parsing
                     Uses stdlib flag package (not go-flags)
  config_test.go

pkg/scraper/
  scraper.go       - BinScraper interface, BinTime struct, NewScraper() registry
  bracknell.go     - BracknellScraper: headless Chrome scraper for Bracknell Forest Council
  wokingham.go     - WokinghamScraper: stub (returns "not implemented")
  scraper_test.go

pkg/clients/
  twilioclient.go  - TwilioClient for sending SMS via Twilio API
  twilioclient_test.go

pkg/schedule/
  schedule.go      - ProjectCollections() — project collections from config rules for a date range
  schedule_test.go

pkg/cache/
  cache.go         - ScraperCache — in-memory TTL cache for scraper results, thread-safe
  cache_test.go

pkg/dateutil/
  dateutil.go      - IsDateMatching(), AsTime(), AsTimeWithMonth(), ParseWeekday(), IsOnWeek()
  dateutil_test.go

pkg/regexp/
  regexp.go        - FindNamedMatches() helper for named capture groups
  regexp_test.go
```

### Application Flow

1. `main()` calls `config.ParseFlags()` then `config.LoadConfig()` to load YAML config
2. `Notifier.Run()` loops over `cfg.Locations`, calling `processLocation()` for each
3. `processLocation()` resolves a scraper via `ScraperFactory`, scrapes bin times, compares against tomorrow's date, and sends SMS if needed
4. SMS messages are prefixed with the location label (e.g., "Home: General Waste, Recycling collection tomorrow")
5. Partial failure: if one location fails, remaining locations still process; exit non-zero if any errors

### Key Interfaces (in cmd/notifier/main.go)

- `BinScraper` - `ScrapeBinTimes(postcode, address) ([]BinTime, error)`
- `SMSClient` - `SendSms(from, to, body, dryRun) error`
- `ScraperFactory` - function type `func(name string) (BinScraper, error)`

### Config Structure (pkg/config)

- `Flags` - CLI flags: ConfigFile (`-c`), DryRun (`-x`), TodayDate (`-d`)
- `Config` - YAML top-level: FromNumber, ToNumber, Locations[], DryRun, TodayDate
- `Location` - Per-location: Label, Scraper, PostCode, AddressCode, CollectionDays ([]CollectionDay)
- `CollectionDay` - Per-schedule: Day (time.Weekday), Types ([]string), EveryNWeeks (int), ReferenceDate (string)

### Adding a New Scraper

1. Create `pkg/scraper/<council>.go` implementing `BinScraper` interface
2. Add a case to the `NewScraper()` switch in `pkg/scraper/scraper.go`
3. No changes needed to `cmd/notifier/main.go`

### MCP Server (cmd/server)

The MCP server exposes bin collection data via the Model Context Protocol over stdio. It provides three tools:

- `get_collections` — Project collections from config schedule rules for a date/range (fast, no Chrome)
- `get_next_collection` — Scrape council website for confirmed next collection dates (cached 6h)
- `list_locations` — List all configured locations and their schedules

Config is loaded via `LoadConfigForMCP()` which skips phone number validation since the MCP server doesn't send SMS.

## Key Dependencies

- `chromedp` - Headless Chrome automation for web scraping
- `twilio-go` - Twilio SDK for SMS
- `mcp-go` - Go MCP SDK for the MCP server
- `yaml.v3` - YAML config file parsing
- `testify` - Test assertions

## GitHub Actions

- **CI** (`.github/workflows/ci.yml`) - Runs on pull requests; builds both notifier and MCP server, runs tests
- **Release** (`.github/workflows/release.yml`) - Triggers on `vX.Y.Z` tags; builds binaries for both notifier and MCP server (linux/amd64, linux/arm64, darwin/arm64), creates a GitHub release with zip artifacts, and pushes Docker images for both to ghcr.io
