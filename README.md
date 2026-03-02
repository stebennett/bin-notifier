# Bin Notifier

A Go application that scrapes bin collection schedules from council websites and sends SMS notifications via Twilio when collections are due tomorrow. Supports multiple locations with pluggable council scrapers.

## Features

- **Multi-location support** — configure multiple addresses in a single YAML config file
- **Pluggable council scrapers** — extensible scraper interface with a registry for adding new councils
- Scrapes bin collection dates using headless Chrome automation
- Sends SMS notifications for upcoming collections via Twilio
- Supports multiple bin types (General Waste, Recycling, Food, Garden)
- Alerts on regular collection days even when no collections are scheduled
- Partial failure handling — continues processing remaining locations if one fails
- SMS messages prefixed with location label for easy identification
- Dry-run mode for testing without sending SMS
- Configurable date override for testing

## Prerequisites

- Go 1.26 or later
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

### Docker

Multi-architecture Docker images are available on GitHub Container Registry:

```bash
docker pull ghcr.io/stebennett/bin-notifier:latest
```

## Configuration

### YAML Config File

Bin Notifier uses a YAML configuration file to define Twilio phone numbers and one or more locations to monitor. Create a `config.yaml` file:

```yaml
from_number: "+441234567890"
to_number: "+447123456789"

locations:
  - label: "Home"
    scraper: "bracknell"
    postcode: "RG12 1AB"
    address_code: "123456"
    collection_days:
      - day: "Tuesday"
        types: ["Recycling", "General Waste"]
      - day: "Friday"
        every_n_weeks: 2
        reference_date: "2026-01-03"
        types: ["Garden Waste"]

  - label: "Office"
    scraper: "wokingham"
    postcode: "RG45 6EF"
    address_code: "120033"
    collection_days:
      - day: "Friday"
        types: ["Household waste", "Food waste"]
      - day: "Friday"
        every_n_weeks: 2
        reference_date: "2026-02-27"
        types: ["Recycling"]
```

#### Location fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | A human-readable name for the location (used in SMS messages and logs) |
| `scraper` | Yes | Which council scraper to use (see available scrapers below) |
| `postcode` | Yes | The postcode to look up on the council website |
| `address_code` | Yes | The address code from the council website |
| `collection_days` | Yes | List of collection day schedules (see below) |

#### Collection day schedule fields

| Field | Required | Description |
|-------|----------|-------------|
| `day` | Yes | Day of the week (e.g. `Monday`, `Tuesday`, ..., `Sunday`) |
| `types` | Yes | List of refuse types collected on this day (e.g. `["Recycling", "General Waste"]`) |
| `every_n_weeks` | No | Collection frequency in weeks (default: `1` for weekly) |
| `reference_date` | When `every_n_weeks > 1` | A known collection date (`YYYY-MM-DD`) used to calculate which weeks are "on". Must fall on the same weekday as `day`. |

#### Available scrapers

| Scraper | Council | Status |
|---------|---------|--------|
| `bracknell` | Bracknell Forest Council | Implemented |
| `wokingham` | Wokingham Borough Council | Implemented |

### Finding Your Address Code

**Bracknell Forest Council:**
1. Visit the [Bracknell Forest Council bin collection page](https://www.bracknell-forest.gov.uk/bins-and-recycling/bin-collection-days)
2. Enter your postcode and select your address
3. The address code appears in the URL or can be found in the page source

**Wokingham Borough Council:**
1. Visit the [Wokingham bin collection page](https://www.wokingham.gov.uk/rubbish-and-recycling/waste-collection/find-your-bin-collection-day)
2. Enter your postcode and click "Find Address"
3. Inspect the address dropdown — each option's `value` attribute is the UPRN (e.g. `120033`). Use this as your `address_code`.

### CLI Flags

| Flag | Short | Env Var | Required | Description |
|------|-------|---------|----------|-------------|
| `--config` | `-c` | `BN_CONFIG_FILE` | Yes | Path to the YAML config file |
| `--dryrun` | `-x` | `BN_DRY_RUN` | No | Run without sending SMS (for testing) |
| `--todaydate` | `-d` | `BN_TODAY_DATE` | No | Override today's date (format: YYYY-MM-DD) |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Yes | Your Twilio account SID |
| `TWILIO_AUTH_TOKEN` | Yes | Your Twilio auth token |
| `BN_FROM_NUMBER` | No | Twilio "from" phone number (overrides `from_number` in config) |
| `BN_TO_NUMBER` | No | Destination phone number (overrides `to_number` in config) |
| `BN_CONFIG_FILE` | No | Path to config file (alternative to `-c` flag) |
| `BN_DRY_RUN` | No | Set to `true` to run without sending SMS |
| `BN_TODAY_DATE` | No | Override today's date (format: YYYY-MM-DD) |

CLI flags take precedence over environment variables. `BN_FROM_NUMBER` and `BN_TO_NUMBER` take precedence over config file values.

## Usage

### Basic Usage

```bash
export TWILIO_ACCOUNT_SID="your_account_sid"
export TWILIO_AUTH_TOKEN="your_auth_token"

./bin-notifier -c config.yaml
```

### Dry Run Mode

Test the scraping without sending SMS:

```bash
./bin-notifier -c config.yaml -x
```

### Override Today's Date

Useful for testing specific scenarios:

```bash
./bin-notifier -c config.yaml -d "2026-01-15"
```

### Docker

Run with Docker by mounting your config file into the container:

```bash
docker run --rm \
  -e TWILIO_ACCOUNT_SID="your_account_sid" \
  -e TWILIO_AUTH_TOKEN="your_auth_token" \
  -v /path/to/config.yaml:/config.yaml:ro \
  ghcr.io/stebennett/bin-notifier:latest \
  -c /config.yaml
```

Dry-run with Docker:

```bash
docker run --rm \
  -e TWILIO_ACCOUNT_SID=test \
  -e TWILIO_AUTH_TOKEN=test \
  -v /path/to/config.yaml:/config.yaml:ro \
  ghcr.io/stebennett/bin-notifier:latest \
  -c /config.yaml -x
```

Build locally:

```bash
docker build -t bin-notifier .
```

### Scheduling with Cron

Run daily at 6 PM to notify about tomorrow's collections:

```bash
0 18 * * * /path/to/bin-notifier -c /path/to/config.yaml
```

## Architecture

```
bin-notifier/
├── cmd/notifier/          # Application entry point
│   ├── main.go            # CLI setup, Notifier orchestration
│   └── main_test.go       # Integration tests
├── pkg/
│   ├── clients/           # External service clients
│   │   ├── twilioclient.go
│   │   └── twilioclient_test.go
│   ├── config/            # Configuration loading
│   │   ├── config.go      # YAML config + CLI flag parsing
│   │   └── config_test.go
│   ├── dateutil/          # Date utilities
│   │   ├── dateutil.go    # Date matching and weekday parsing
│   │   └── dateutil_test.go
│   ├── regexp/            # Regex utilities
│   │   ├── regexp.go
│   │   └── regexp_test.go
│   └── scraper/           # Web scraping logic
│       ├── scraper.go     # BinScraper interface + registry
│       ├── scraper_test.go
│       ├── bracknell.go   # Bracknell Forest Council scraper
│       └── wokingham.go   # Wokingham Borough Council scraper
└── .github/workflows/     # CI/CD pipelines
    ├── ci.yml             # Build and test on PRs
    └── release.yml        # Release automation
```

### Application Flow

1. **Configuration** — Parse CLI flags, then load the YAML config file containing phone numbers and locations
2. **Location loop** — For each configured location:
   1. Look up the scraper by name from the registry
   2. Use headless Chrome to navigate the council website and extract collection dates
   3. Compare scraped dates against tomorrow's date
3. **Notification** — Send SMS via Twilio for each location where collections are due or it is a regular collection day with no scheduled collections
4. **Partial Failure** — If one location fails, processing continues for remaining locations; exits non-zero if any location had errors

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
| [yaml.v3](https://gopkg.in/yaml.v3) | YAML config file parsing |
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

This triggers automated builds for all supported platforms, creates a GitHub release with downloadable zip archives, and pushes Docker images to GitHub Container Registry.

## License

See [LICENSE](LICENSE) for details.
