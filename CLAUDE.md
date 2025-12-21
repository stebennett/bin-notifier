# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the application
go build -o bin-notifier ./cmd/notifier

# Run the application (requires environment variables or flags)
go run cmd/notifier/main.go -p <POSTCODE> -a <ADDRESS_CODE> -f <FROM_NUMBER> -n <TO_NUMBER> -r <COLLECTION_DAY>

# Run with dry-run mode (no SMS sent)
go run cmd/notifier/main.go -p <POSTCODE> -a <ADDRESS_CODE> -f <FROM_NUMBER> -n <TO_NUMBER> -r <COLLECTION_DAY> -x

# Run with a specific date (for testing date logic)
go run cmd/notifier/main.go -p <POSTCODE> -a <ADDRESS_CODE> -f <FROM_NUMBER> -n <TO_NUMBER> -r <COLLECTION_DAY> -d 2024-01-15

# Run tests
go test ./...

# Run a specific test
go test -run TestParseNextCollectionTime ./pkg/scraper/
```

## Prerequisites

- Google Chrome or Chromium (required for headless web scraping)

## Required Environment Variables (for Twilio)

- `TWILIO_ACCOUNT_SID` - Twilio account SID
- `TWILIO_AUTH_TOKEN` - Twilio auth token

## Architecture

This is a Go application that scrapes bin collection schedules from the Bracknell Forest Council website and sends SMS notifications via Twilio when collections are due tomorrow.

**Flow:**
1. `cmd/notifier/main.go` - Entry point that orchestrates the workflow
2. `pkg/scraper/` - Uses chromedp (headless Chrome) to scrape collection dates from the council website
3. `pkg/clients/` - Twilio client for sending SMS notifications
4. `pkg/config/` - CLI argument parsing using go-flags
5. `pkg/dateutil/` - Date comparison and parsing utilities
6. `pkg/regexp/` - Helper for extracting named regex groups

**Key dependencies:**
- `chromedp` - Headless Chrome automation for web scraping
- `twilio-go` - Twilio SDK for SMS
- `go-flags` - CLI argument parsing
- `testify` - Test assertions

## GitHub Actions

- **CI** (`.github/workflows/ci.yml`) - Runs on pull requests; builds and tests the application
- **Release** (`.github/workflows/release.yml`) - Triggers on `vX.Y.Z` tags; builds for linux/amd64, linux/arm64, darwin/arm64 and creates a GitHub release with zip artifacts
