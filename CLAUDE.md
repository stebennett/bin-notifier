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

# Build Docker image
docker build -t bin-notifier .

# Run with Docker (dry-run mode)
docker run --rm \
  -e TWILIO_ACCOUNT_SID=test \
  -e TWILIO_AUTH_TOKEN=test \
  -v /path/to/config.yaml:/config.yaml:ro \
  bin-notifier -c /config.yaml -x
```

## Prerequisites

- Google Chrome or Chromium (required for headless web scraping)

## Environment Variables

**Twilio (required):**
- `TWILIO_ACCOUNT_SID` - Twilio account SID
- `TWILIO_AUTH_TOKEN` - Twilio auth token

**Application config (alternative to CLI flags):**
- `BN_CONFIG_FILE` - Path to YAML config file
- `BN_DRY_RUN` - Set to `true` for dry-run mode
- `BN_TODAY_DATE` - Override today's date (YYYY-MM-DD)

CLI flags take precedence over environment variables.

## Architecture

This is a Go application that scrapes bin collection schedules from the Bracknell Forest Council website and sends SMS notifications via Twilio when collections are due tomorrow.

**Flow:**
1. `cmd/notifier/main.go` - Entry point, parses flags, loads config, loops over locations
2. `pkg/scraper/` - Scraper interface + registry + council-specific implementations (e.g., Bracknell)
3. `pkg/clients/` - Twilio client for sending SMS notifications
4. `pkg/config/` - YAML config file loading + CLI flag parsing
5. `pkg/dateutil/` - Date comparison, parsing, and weekday name utilities
6. `pkg/regexp/` - Helper for extracting named regex groups

**Key dependencies:**
- `chromedp` - Headless Chrome automation for web scraping
- `twilio-go` - Twilio SDK for SMS
- `yaml.v3` - YAML config file parsing
- `testify` - Test assertions

## GitHub Actions

- **CI** (`.github/workflows/ci.yml`) - Runs on pull requests; builds and tests the application
- **Release** (`.github/workflows/release.yml`) - Triggers on `vX.Y.Z` tags; builds binaries for linux/amd64, linux/arm64, darwin/arm64, creates a GitHub release with zip artifacts, and pushes Docker images to ghcr.io
