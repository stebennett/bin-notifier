# Bin Notifier

A Go application that scrapes bin collection schedules from the Bracknell Forest Council website and sends SMS notifications via Twilio when collections are due tomorrow.

## Features

- Scrapes bin collection dates using headless Chrome automation
- Sends SMS notifications for upcoming collections via Twilio
- Supports multiple bin types (General Waste, Recycling, Food, Garden)
- Alerts on regular collection days even when no collections are scheduled
- Dry-run mode for testing without sending SMS
- Configurable date override for testing

## Prerequisites

- Go 1.20 or later
- Google Chrome or Chromium (for headless scraping)
- Twilio account with SMS capabilities

## Installation

### From Source

```bash
git clone https://github.com/stebennett/bin-notifier.git
cd bin-notifier
go build -o bin-notifier ./cmd/notifier
```

### From Releases

Download pre-built binaries from the [GitHub Releases](https://github.com/stebennett/bin-notifier/releases) page. Available for:
- Linux (amd64, arm64)
- macOS (arm64)

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Yes | Your Twilio account SID |
| `TWILIO_AUTH_TOKEN` | Yes | Your Twilio auth token |

### Command Line Flags

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--postcode` | `-p` | Yes | The postcode to scrape bin times for |
| `--addressCode` | `-a` | Yes | The address code from the council website |
| `--regularcollectionday` | `-r` | Yes | Regular collection day (0=Sunday, 1=Monday, ..., 6=Saturday) |
| `--fromnumber` | `-f` | Yes | Twilio phone number to send SMS from |
| `--tonumber` | `-n` | Yes | Phone number to send SMS notifications to |
| `--dryrun` | `-x` | No | Run without sending SMS (for testing) |
| `--todaydate` | `-d` | No | Override today's date (format: YYYY-MM-DD) |

### Finding Your Address Code

1. Visit the [Bracknell Forest Council bin collection page](https://www.bracknell-forest.gov.uk/bins-and-recycling/bin-collection-days)
2. Enter your postcode and select your address
3. The address code appears in the URL or can be found in the page source

## Usage

### Basic Usage

```bash
export TWILIO_ACCOUNT_SID="your_account_sid"
export TWILIO_AUTH_TOKEN="your_auth_token"

./bin-notifier \
  -p "RG12 1AB" \
  -a "123456" \
  -r 2 \
  -f "+441234567890" \
  -n "+447123456789"
```

### Dry Run Mode

Test the scraping without sending SMS:

```bash
./bin-notifier \
  -p "RG12 1AB" \
  -a "123456" \
  -r 2 \
  -f "+441234567890" \
  -n "+447123456789" \
  -x
```

### Override Today's Date

Useful for testing specific scenarios:

```bash
./bin-notifier \
  -p "RG12 1AB" \
  -a "123456" \
  -r 2 \
  -f "+441234567890" \
  -n "+447123456789" \
  -d "2024-01-15"
```

### Scheduling with Cron

Run daily at 6 PM to notify about tomorrow's collections:

```bash
0 18 * * * /path/to/bin-notifier -p "RG12 1AB" -a "123456" -r 2 -f "+441234567890" -n "+447123456789"
```

## Architecture

```
bin-notifier/
├── cmd/notifier/          # Application entry point
│   ├── main.go            # CLI setup and orchestration
│   └── main_test.go       # Integration tests
├── pkg/
│   ├── clients/           # External service clients
│   │   ├── twilioclient.go      # Twilio SMS client
│   │   └── twilioclient_test.go
│   ├── config/            # CLI configuration parsing
│   │   ├── config.go
│   │   └── config_test.go
│   ├── dateutil/          # Date utilities
│   │   ├── dateutil.go
│   │   └── dateutil_test.go
│   ├── regexp/            # Regex utilities
│   │   ├── regexp.go
│   │   └── regexp_test.go
│   └── scraper/           # Web scraping logic
│       ├── scraper.go
│       └── scraper_test.go
└── .github/workflows/     # CI/CD pipelines
    ├── ci.yml             # Build and test on PRs
    └── release.yml        # Release automation
```

### Application Flow

1. **Configuration** - Parse CLI arguments and environment variables
2. **Scraping** - Use headless Chrome to navigate the council website and extract collection dates
3. **Date Matching** - Compare scraped dates against tomorrow's date
4. **Notification** - Send SMS via Twilio if collections are due or it's a regular collection day with no scheduled collections

## Development

### Building

```bash
go build -o bin-notifier ./cmd/notifier
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/scraper/

# Run specific test
go test -run TestParseNextCollectionTime ./pkg/scraper/

# Run with coverage
go test -cover ./...
```

### Key Dependencies

| Package | Purpose |
|---------|---------|
| [chromedp](https://github.com/chromedp/chromedp) | Headless Chrome automation |
| [twilio-go](https://github.com/twilio/twilio-go) | Twilio SDK for SMS |
| [go-flags](https://github.com/jessevdk/go-flags) | CLI argument parsing |
| [testify](https://github.com/stretchr/testify) | Test assertions |

## CI/CD

### Continuous Integration

Pull requests automatically trigger:
- Build verification
- Test suite execution

### Releases

To create a release, push a semantic version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers automated builds for all supported platforms and creates a GitHub release with downloadable zip archives.

## License

See [LICENSE](LICENSE) for details.
