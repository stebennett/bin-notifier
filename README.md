# Bin Notifier

A Go application that scrapes bin collection schedules from council websites and sends SMS notifications via Twilio when collections are due tomorrow. Supports multiple locations with pluggable council scrapers.

## Features

- Scrapes bin collection dates using headless Chrome automation
- Sends SMS notifications for upcoming collections via Twilio
- **Multi-location support** — monitor multiple addresses with a single config file
- **Pluggable scrapers** — extensible scraper registry for different council websites
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

Create a `config.yaml` file with your locations and notification settings:

```yaml
from_number: "+441234567890"
to_number: "+447123456789"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "123456"
    collection_day: tuesday
  - label: Office
    scraper: bracknell
    postcode: "RG42 2XY"
    address_code: "789012"
    collection_day: thursday
```

#### Config Fields

| Field | Required | Description |
|-------|----------|-------------|
| `from_number` | Yes | Twilio phone number to send SMS from |
| `to_number` | Yes | Phone number to send SMS notifications to |
| `locations` | Yes | List of locations to monitor (at least one) |

#### Location Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Display name for the location (used in SMS messages) |
| `scraper` | Yes | Scraper to use (e.g., `bracknell`) |
| `postcode` | Yes | Postcode to look up |
| `address_code` | Yes | Address code from the council website |
| `collection_day` | Yes | Regular collection day name (e.g., `tuesday`) |

### Available Scrapers

| Name | Council |
|------|---------|
| `bracknell` | Bracknell Forest Council |

### Environment Variables

#### Twilio Credentials

| Variable | Required | Description |
|----------|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Yes | Your Twilio account SID |
| `TWILIO_AUTH_TOKEN` | Yes | Your Twilio auth token |

#### Application Configuration

| Variable | Description |
|----------|-------------|
| `BN_CONFIG_FILE` | Path to YAML config file (alternative to `-c` flag) |
| `BN_DRY_RUN` | Set to `true` to run without sending SMS |
| `BN_TODAY_DATE` | Override today's date (format: YYYY-MM-DD) |

### Command Line Flags

| Flag | Short | Env Var | Required | Description |
|------|-------|---------|----------|-------------|
| `--config` | `-c` | `BN_CONFIG_FILE` | Yes | Path to YAML config file |
| `--dryrun` | `-x` | `BN_DRY_RUN` | No | Run without sending SMS |
| `--todaydate` | `-d` | `BN_TODAY_DATE` | No | Override today's date (YYYY-MM-DD) |

CLI flags take precedence over environment variables.

### Finding Your Address Code

1. Visit the [Bracknell Forest Council bin collection page](https://www.bracknell-forest.gov.uk/bins-and-recycling/bin-collection-days)
2. Enter your postcode and select your address
3. The address code appears in the URL or can be found in the page source

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
./bin-notifier -c config.yaml -d "2024-01-15"
```

### Docker

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

### Scheduling with Cron

Run daily at 6 PM to notify about tomorrow's collections:

```bash
0 18 * * * /path/to/bin-notifier -c /path/to/config.yaml
```

## Architecture

```
bin-notifier/
├── cmd/notifier/          # Application entry point
│   ├── main.go            # CLI setup and orchestration
│   └── main_test.go       # Integration tests
├── pkg/
│   ├── clients/           # External service clients
│   │   ├── twilioclient.go
│   │   └── twilioclient_test.go
│   ├── config/            # YAML config + CLI flag parsing
│   │   ├── config.go
│   │   └── config_test.go
│   ├── dateutil/          # Date utilities
│   │   ├── dateutil.go
│   │   └── dateutil_test.go
│   ├── regexp/            # Regex utilities
│   │   ├── regexp.go
│   │   └── regexp_test.go
│   └── scraper/           # Web scraping logic
│       ├── scraper.go     # Interface + registry
│       ├── bracknell.go   # Bracknell Forest Council scraper
│       └── scraper_test.go
└── .github/workflows/     # CI/CD pipelines
    ├── ci.yml             # Build and test on PRs
    └── release.yml        # Release automation
```

### Application Flow

1. **Configuration** — Parse CLI flags, load YAML config file
2. **Per-Location Processing** — For each location in the config:
   a. Resolve the scraper by name from the registry
   b. Use headless Chrome to scrape collection dates from the council website
   c. Compare scraped dates against tomorrow's date
   d. Send SMS via Twilio if collections are due or it's a regular collection day with no scheduled collections
3. **Partial Failure** — If one location fails, processing continues for remaining locations; exits non-zero if any location had errors

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
