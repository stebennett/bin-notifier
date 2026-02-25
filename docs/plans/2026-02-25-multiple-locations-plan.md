# Multiple Locations Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor bin-notifier to support multiple locations with per-location scrapers, YAML config file, and partial failure handling.

**Architecture:** Replace go-flags CLI parsing with YAML config file + minimal flag parsing. Refactor the scraper package into an interface with a registry of council-specific implementations. Notifier loops over locations, sending one SMS per location, continuing on partial failures.

**Tech Stack:** Go 1.26, gopkg.in/yaml.v3 (already indirect dep), chromedp, twilio-go, testify

---

### Task 1: Add day name parsing to dateutil

**Files:**
- Modify: `pkg/dateutil/dateutil.go`
- Modify: `pkg/dateutil/dateutil_test.go`

**Step 1: Write the failing tests**

Add to `pkg/dateutil/dateutil_test.go`:

```go
func TestParseWeekday(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Weekday
		hasError bool
	}{
		{name: "lowercase monday", input: "monday", expected: time.Monday},
		{name: "uppercase TUESDAY", input: "TUESDAY", expected: time.Tuesday},
		{name: "mixed case Wednesday", input: "Wednesday", expected: time.Wednesday},
		{name: "thursday", input: "thursday", expected: time.Thursday},
		{name: "friday", input: "friday", expected: time.Friday},
		{name: "saturday", input: "saturday", expected: time.Saturday},
		{name: "sunday", input: "sunday", expected: time.Sunday},
		{name: "invalid day", input: "notaday", hasError: true},
		{name: "empty string", input: "", hasError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseWeekday(test.input)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/dateutil/ -run TestParseWeekday -v`
Expected: FAIL — `ParseWeekday` not defined

**Step 3: Write minimal implementation**

Add to `pkg/dateutil/dateutil.go`:

```go
import (
	"fmt"
	"strings"
	"time"
)

func ParseWeekday(s string) (time.Weekday, error) {
	days := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}
	day, ok := days[strings.ToLower(s)]
	if !ok {
		return 0, fmt.Errorf("invalid weekday: %q", s)
	}
	return day, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/dateutil/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add pkg/dateutil/dateutil.go pkg/dateutil/dateutil_test.go
git commit -m "feat: add ParseWeekday to dateutil for human-readable day names"
```

---

### Task 2: Replace config with YAML config file parsing

**Files:**
- Rewrite: `pkg/config/config.go`
- Rewrite: `pkg/config/config_test.go`

**Step 1: Write the failing tests**

Replace `pkg/config/config_test.go` entirely:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestLoadConfig_ValidMultipleLocations(t *testing.T) {
	path := writeConfigFile(t, `
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
`)

	cfg, err := LoadConfig(path)

	assert.NoError(t, err)
	assert.Equal(t, "+441234567890", cfg.FromNumber)
	assert.Equal(t, "+449876543210", cfg.ToNumber)
	assert.Len(t, cfg.Locations, 2)

	assert.Equal(t, "Home", cfg.Locations[0].Label)
	assert.Equal(t, "bracknell", cfg.Locations[0].Scraper)
	assert.Equal(t, "RG12 1AB", cfg.Locations[0].PostCode)
	assert.Equal(t, "12345", cfg.Locations[0].AddressCode)
	assert.Equal(t, time.Tuesday, cfg.Locations[0].CollectionDay)

	assert.Equal(t, "Office", cfg.Locations[1].Label)
	assert.Equal(t, "wokingham", cfg.Locations[1].Scraper)
}

func TestLoadConfig_CaseInsensitiveDay(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: WEDNESDAY
`)

	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, time.Wednesday, cfg.Locations[0].CollectionDay)
}

func TestLoadConfig_InvalidDay(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: notaday
`)

	_, err := LoadConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid weekday")
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		errText string
	}{
		{
			name: "missing from_number",
			yaml: `
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "from_number is required",
		},
		{
			name: "missing to_number",
			yaml: `
from_number: "+441234567890"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "to_number is required",
		},
		{
			name: "no locations",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations: []`,
			errText: "at least one location is required",
		},
		{
			name: "missing label",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "label is required",
		},
		{
			name: "missing scraper",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "scraper is required",
		},
		{
			name: "missing postcode",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    address_code: "12345"
    collection_day: tuesday`,
			errText: "postcode is required",
		},
		{
			name: "missing address_code",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    collection_day: tuesday`,
			errText: "address_code is required",
		},
		{
			name: "missing collection_day",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"`,
			errText: "collection_day is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfigFile(t, test.yaml)
			_, err := LoadConfig(path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.errText)
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := writeConfigFile(t, `{{{invalid yaml`)
	_, err := LoadConfig(path)
	assert.Error(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/ -v`
Expected: FAIL — `LoadConfig` not defined

**Step 3: Write minimal implementation**

Replace `pkg/config/config.go` entirely:

```go
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"gopkg.in/yaml.v3"
)

type Location struct {
	Label         string       `yaml:"label"`
	Scraper       string       `yaml:"scraper"`
	PostCode      string       `yaml:"postcode"`
	AddressCode   string       `yaml:"address_code"`
	CollectionDay time.Weekday `yaml:"-"`
	RawDay        string       `yaml:"collection_day"`
}

type Config struct {
	FromNumber string     `yaml:"from_number"`
	ToNumber   string     `yaml:"to_number"`
	Locations  []Location `yaml:"locations"`
	DryRun     bool       `yaml:"-"`
	TodayDate  string     `yaml:"-"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if err := validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.FromNumber == "" {
		return fmt.Errorf("from_number is required")
	}
	if cfg.ToNumber == "" {
		return fmt.Errorf("to_number is required")
	}
	if len(cfg.Locations) == 0 {
		return fmt.Errorf("at least one location is required")
	}
	for i := range cfg.Locations {
		loc := &cfg.Locations[i]
		if loc.Label == "" {
			return fmt.Errorf("location %d: label is required", i+1)
		}
		if loc.Scraper == "" {
			return fmt.Errorf("location %d: scraper is required", i+1)
		}
		if loc.PostCode == "" {
			return fmt.Errorf("location %d: postcode is required", i+1)
		}
		if loc.AddressCode == "" {
			return fmt.Errorf("location %d: address_code is required", i+1)
		}
		if loc.RawDay == "" {
			return fmt.Errorf("location %d: collection_day is required", i+1)
		}
		day, err := dateutil.ParseWeekday(loc.RawDay)
		if err != nil {
			return fmt.Errorf("location %d: %w", i+1, err)
		}
		loc.CollectionDay = day
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/config/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: replace CLI flag config with YAML config file parsing"
```

---

### Task 3: Add CLI flag parsing for config path, dry-run, and date override

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

**Step 1: Write the failing tests**

Add to `pkg/config/config_test.go`:

```go
func TestParseFlags_AllFlags(t *testing.T) {
	flags, err := ParseFlags([]string{"-c", "/path/to/config.yaml", "-x", "-d", "2026-01-15"})
	assert.NoError(t, err)
	assert.Equal(t, "/path/to/config.yaml", flags.ConfigFile)
	assert.True(t, flags.DryRun)
	assert.Equal(t, "2026-01-15", flags.TodayDate)
}

func TestParseFlags_ConfigOnly(t *testing.T) {
	flags, err := ParseFlags([]string{"-c", "/path/to/config.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "/path/to/config.yaml", flags.ConfigFile)
	assert.False(t, flags.DryRun)
	assert.Equal(t, "", flags.TodayDate)
}

func TestParseFlags_EnvVarFallback(t *testing.T) {
	orig := os.Getenv("BN_CONFIG_FILE")
	defer func() {
		if orig == "" {
			os.Unsetenv("BN_CONFIG_FILE")
		} else {
			os.Setenv("BN_CONFIG_FILE", orig)
		}
	}()

	os.Setenv("BN_CONFIG_FILE", "/env/config.yaml")
	flags, err := ParseFlags([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "/env/config.yaml", flags.ConfigFile)
}

func TestParseFlags_MissingConfigFile(t *testing.T) {
	orig := os.Getenv("BN_CONFIG_FILE")
	defer func() {
		if orig == "" {
			os.Unsetenv("BN_CONFIG_FILE")
		} else {
			os.Setenv("BN_CONFIG_FILE", orig)
		}
	}()
	os.Unsetenv("BN_CONFIG_FILE")

	_, err := ParseFlags([]string{})
	assert.Error(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/ -run TestParseFlags -v`
Expected: FAIL — `ParseFlags` not defined

**Step 3: Write minimal implementation**

Add to `pkg/config/config.go`:

```go
import flags "github.com/jessevdk/go-flags"

type Flags struct {
	ConfigFile string `short:"c" long:"config" env:"BN_CONFIG_FILE" description:"Path to config file" required:"true"`
	DryRun     bool   `short:"x" long:"dryrun" env:"BN_DRY_RUN" description:"Run without sending SMS"`
	TodayDate  string `short:"d" long:"todaydate" env:"BN_TODAY_DATE" description:"Override today's date (YYYY-MM-DD)"`
}

func ParseFlags(args []string) (Flags, error) {
	var f Flags
	parser := flags.NewParser(&f, flags.Default)
	_, err := parser.ParseArgs(args)
	if err != nil {
		return f, err
	}
	return f, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/config/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: add CLI flag parsing for config path, dry-run, and date override"
```

---

### Task 4: Refactor scraper into interface + registry + Bracknell implementation

**Files:**
- Rewrite: `pkg/scraper/scraper.go` → shared types + registry
- Create: `pkg/scraper/bracknell.go` → Bracknell-specific scraping code
- Modify: `pkg/scraper/scraper_test.go`

**Step 1: Write the failing test for the registry**

Add to `pkg/scraper/scraper_test.go`:

```go
func TestNewScraper_Bracknell(t *testing.T) {
	s, err := NewScraper("bracknell")
	assert.NoError(t, err)
	assert.IsType(t, &BracknellScraper{}, s)
}

func TestNewScraper_UnknownReturnsError(t *testing.T) {
	_, err := NewScraper("unknown_council")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown scraper")
}

func TestNewScraper_CaseInsensitive(t *testing.T) {
	s, err := NewScraper("Bracknell")
	assert.NoError(t, err)
	assert.IsType(t, &BracknellScraper{}, s)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/scraper/ -run TestNewScraper -v`
Expected: FAIL — `NewScraper`, `BracknellScraper` not defined

**Step 3: Refactor the scraper package**

Create `pkg/scraper/bracknell.go` — move the `BinTimesScraper` struct, `ScrapeBinTimes`, and `parseNextCollectionTime` from `scraper.go` into this file, renaming the struct to `BracknellScraper`:

```go
package scraper

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
	regexputil "github.com/stebennett/bin-notifier/pkg/regexp"
)

type BracknellScraper struct{}

func (s *BracknellScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	if len(postCode) == 0 {
		return []BinTime{}, errors.New("no postcode specified")
	}
	if len(addressCode) == 0 {
		return []BinTime{}, errors.New("no address specified")
	}

	log.Printf("creating temp user data dir")
	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	log.Printf("setting chrome defaults")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", true),
	)

	log.Printf("creating chrome context")
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	log.Printf("creating logger")
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	log.Printf("creating timeout")
	taskCtx, cancel = context.WithTimeout(taskCtx, 60*time.Second)
	defer cancel()

	log.Printf("running task")
	collectionTimes := make([]string, 4)

	err = chromedp.Run(taskCtx,
		chromedp.Navigate("https://selfservice.mybfc.bracknell-forest.gov.uk/w/webpage/waste-collection-days"),

		chromedp.WaitVisible(`//a[text()="Accept all cookies"]`),
		chromedp.Click(`//a[text()="Accept all cookies"]`),
		chromedp.WaitNotVisible(`//a[text()="Accept all cookies"]`),

		chromedp.SendKeys(`input[type="text"]`, postCode),
		chromedp.Sleep(2*time.Second),

		chromedp.SendKeys(`input[type="text"]`, kb.Enter),
		chromedp.Sleep(2*time.Second),

		chromedp.WaitVisible(`//select`),
		chromedp.SetValue(`//select`, addressCode),
		chromedp.Sleep(2*time.Second),
		chromedp.EvaluateAsDevTools(`document.querySelector("select").dispatchEvent(new Event("change"))`, nil),

		chromedp.WaitVisible(`//h2[@class="collectionHeading"]`),

		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[1]/tr/td[2]`, &collectionTimes[0], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[2]/tr/td[2]`, &collectionTimes[1], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[3]/tr/td[2]`, &collectionTimes[2], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[4]/tr/td[2]`, &collectionTimes[3], chromedp.NodeVisible, chromedp.BySearch),
	)

	if err != nil {
		return []BinTime{}, err
	}

	log.Printf("bin times collected. parsing to extract.")

	binTimes := make([]BinTime, len(collectionTimes))
	for i, t := range collectionTimes {
		binTimes[i], err = parseBracknellCollectionTime(t)
		if err != nil {
			return binTimes, err
		}
	}

	return binTimes, nil
}

func parseBracknellCollectionTime(times string) (BinTime, error) {
	t := strings.Split(times, "\n")

	exp := `Your next (?P<BinType>[a-z\s]+) collection is [A-Za-z]+ (?P<Date>\d+) (?P<Month>[A-Za-z]+) (?P<Year>\d{4})`
	re := regexp.MustCompile(exp)

	matches := regexputil.FindNamedMatches(re, t[0])
	if len(matches) != 4 {
		return BinTime{}, errors.New("failed to parse next collection time")
	}

	day, _ := strconv.Atoi(matches["Date"])
	year, _ := strconv.Atoi(matches["Year"])

	return BinTime{matches["BinType"], dateutil.AsTimeWithMonth(day, matches["Month"], year)}, nil
}
```

Rewrite `pkg/scraper/scraper.go` to contain only shared types and the registry:

```go
package scraper

import (
	"fmt"
	"strings"
	"time"
)

type BinTime struct {
	Type           string
	CollectionTime time.Time
}

type BinScraper interface {
	ScrapeBinTimes(postcode string, addressCode string) ([]BinTime, error)
}

func NewScraper(name string) (BinScraper, error) {
	switch strings.ToLower(name) {
	case "bracknell":
		return &BracknellScraper{}, nil
	case "wokingham":
		return &WokinghamScraper{}, nil
	default:
		return nil, fmt.Errorf("unknown scraper: %q", name)
	}
}
```

**Step 4: Update existing tests**

In `pkg/scraper/scraper_test.go`, rename `parseNextCollectionTime` references to `parseBracknellCollectionTime`. Also rename `NewBinTimesScraper` references to `&BracknellScraper{}` in `TestScrapeBinTimes_ValidationErrors`:

```go
// In each test case calling parseNextCollectionTime, change to:
actual, err := parseBracknellCollectionTime(test.input)

// In TestScrapeBinTimes_ValidationErrors, change:
scraper := &BracknellScraper{}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./pkg/scraper/ -v`
Expected: ALL PASS (will fail until Task 5 creates the Wokingham stub)

**Step 6: Commit**

```bash
git add pkg/scraper/scraper.go pkg/scraper/bracknell.go pkg/scraper/scraper_test.go
git commit -m "refactor: split scraper into interface, registry, and Bracknell implementation"
```

---

### Task 5: Add stubbed Wokingham scraper

**Files:**
- Create: `pkg/scraper/wokingham.go`
- Modify: `pkg/scraper/scraper_test.go`

**Step 1: Write the failing test**

Add to `pkg/scraper/scraper_test.go`:

```go
func TestNewScraper_Wokingham(t *testing.T) {
	s, err := NewScraper("wokingham")
	assert.NoError(t, err)
	assert.IsType(t, &WokinghamScraper{}, s)
}

func TestWokinghamScraper_ReturnsNotImplemented(t *testing.T) {
	s := &WokinghamScraper{}
	_, err := s.ScrapeBinTimes("RG41 1AA", "12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/scraper/ -run TestWokingham -v`
Expected: FAIL — `WokinghamScraper` not defined

**Step 3: Write minimal implementation**

Create `pkg/scraper/wokingham.go`:

```go
package scraper

import "fmt"

type WokinghamScraper struct{}

func (s *WokinghamScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	return nil, fmt.Errorf("wokingham scraper not implemented")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/scraper/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add pkg/scraper/wokingham.go pkg/scraper/scraper_test.go
git commit -m "feat: add stubbed Wokingham scraper"
```

---

### Task 6: Refactor Notifier to support multiple locations

**Files:**
- Rewrite: `cmd/notifier/main.go`
- Rewrite: `cmd/notifier/main_test.go`

**Step 1: Write the failing tests**

Replace `cmd/notifier/main_test.go` entirely:

```go
package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/scraper"
	"github.com/stretchr/testify/assert"
)

type mockScraper struct {
	binTimes []scraper.BinTime
	err      error
}

func (m *mockScraper) ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error) {
	return m.binTimes, m.err
}

type mockSMSClient struct {
	messages []smsMessage
	err      error
}

type smsMessage struct {
	from, to, body string
	dryRun         bool
}

func (m *mockSMSClient) SendSms(from string, to string, body string, dryRun bool) error {
	m.messages = append(m.messages, smsMessage{from, to, body, dryRun})
	return m.err
}

type mockScraperFactory struct {
	scrapers map[string]*mockScraper
}

func (f *mockScraperFactory) NewScraper(name string) (BinScraper, error) {
	s, ok := f.scrapers[name]
	if !ok {
		return nil, errors.New("unknown scraper: " + name)
	}
	return s, nil
}

func createTestConfig() config.Config {
	return config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:         "Home",
				Scraper:       "bracknell",
				PostCode:      "RG12 1AB",
				AddressCode:   "12345",
				CollectionDay: time.Tuesday,
			},
		},
	}
}

func TestNotifier_MultipleLocationsWithCollections(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	tomorrow := today.AddDate(0, 0, 1)                       // Tuesday

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
			}},
			"wokingham": {binTimes: []scraper.BinTime{
				{Type: "Food", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "RG12 1AB", AddressCode: "12345", CollectionDay: time.Tuesday},
			{Label: "Office", Scraper: "wokingham", PostCode: "RG42 2XY", AddressCode: "67890", CollectionDay: time.Thursday},
		},
	}

	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Len(t, mockSMS.messages, 2)
	assert.Contains(t, mockSMS.messages[0].body, "Home")
	assert.Contains(t, mockSMS.messages[0].body, "Recycling")
	assert.Contains(t, mockSMS.messages[1].body, "Office")
	assert.Contains(t, mockSMS.messages[1].body, "Food")
}

func TestNotifier_SingleLocationWithCollections(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow},
				{Type: "Recycling", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].SMSSent)
	assert.Contains(t, mockSMS.messages[0].body, "Home")
	assert.Contains(t, mockSMS.messages[0].body, "General Waste")
	assert.Contains(t, mockSMS.messages[0].body, "Recycling")
}

func TestNotifier_RegularDayNoCollections(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: nextWeek},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.Locations[0].CollectionDay = time.Tuesday
	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].SMSSent)
	assert.Contains(t, mockSMS.messages[0].body, "Home")
	assert.Contains(t, mockSMS.messages[0].body, "no collections")
}

func TestNotifier_NoCollectionsNotRegularDay(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	nextWeek := today.AddDate(0, 0, 7)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: nextWeek},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.Locations[0].CollectionDay = time.Wednesday
	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].SMSSent)
	assert.Len(t, mockSMS.messages, 0)
}

func TestNotifier_PartialFailure_ContinuesForSuccessful(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {err: errors.New("scraper failed")},
			"wokingham": {binTimes: []scraper.BinTime{
				{Type: "Food", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "RG12 1AB", AddressCode: "12345", CollectionDay: time.Tuesday},
			{Label: "Office", Scraper: "wokingham", PostCode: "RG42 2XY", AddressCode: "67890", CollectionDay: time.Thursday},
		},
	}

	results, err := notifier.Run(cfg)

	assert.Error(t, err) // aggregate error
	assert.Len(t, results, 2)
	assert.NotNil(t, results[0].Error)
	assert.True(t, results[1].SMSSent)
	assert.Contains(t, mockSMS.messages[0].body, "Office")
}

func TestNotifier_UsesTodayDateOverride(t *testing.T) {
	clock := func() time.Time { return time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) }
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Jan 15 + 1

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          clock,
	}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15"
	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.True(t, results[0].SMSSent)
}

func TestNotifier_DryRunPassedToSmsClient(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.DryRun = true
	_, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.True(t, mockSMS.messages[0].dryRun)
}

func TestNotifier_LabelAppearsInMessage(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.Locations[0].Label = "My House"
	results, err := notifier.Run(cfg)

	assert.NoError(t, err)
	assert.Contains(t, results[0].Message, "My House")
	assert.Contains(t, mockSMS.messages[0].body, "My House")
}

func TestNotifier_SMSErrorForOneLocation(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	factory := &mockScraperFactory{
		scrapers: map[string]*mockScraper{
			"bracknell": {binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
			}},
		},
	}
	mockSMS := &mockSMSClient{err: errors.New("SMS failed")}

	notifier := &Notifier{
		ScraperFactory: factory,
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results, err := notifier.Run(cfg)

	assert.Error(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].Error)
	assert.False(t, results[0].SMSSent)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/notifier/ -v`
Expected: FAIL — new interfaces/types not defined

**Step 3: Write the implementation**

Rewrite `cmd/notifier/main.go`:

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/stebennett/bin-notifier/pkg/clients"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"github.com/stebennett/bin-notifier/pkg/scraper"
)

type BinScraper interface {
	ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error)
}

type SMSClient interface {
	SendSms(from string, to string, body string, dryRun bool) error
}

type ScraperFactory interface {
	NewScraper(name string) (BinScraper, error)
}

type defaultScraperFactory struct{}

func (f *defaultScraperFactory) NewScraper(name string) (BinScraper, error) {
	return scraper.NewScraper(name)
}

type Notifier struct {
	ScraperFactory ScraperFactory
	SMSClient      SMSClient
	Clock          func() time.Time
}

type NotificationResult struct {
	Label       string
	Collections []string
	SMSSent     bool
	Message     string
	Error       error
}

func (n *Notifier) Run(cfg config.Config) ([]NotificationResult, error) {
	today := n.Clock()
	if cfg.TodayDate != "" {
		var err error
		today, err = time.Parse("2006-01-02", cfg.TodayDate)
		if err != nil {
			return nil, fmt.Errorf("invalid today date: %w", err)
		}
	}
	tomorrow := today.AddDate(0, 0, 1)

	results := make([]NotificationResult, len(cfg.Locations))
	var errs []error

	for i, loc := range cfg.Locations {
		results[i].Label = loc.Label
		result := n.processLocation(loc, tomorrow, cfg.FromNumber, cfg.ToNumber, cfg.DryRun)
		results[i] = result
		if result.Error != nil {
			errs = append(errs, fmt.Errorf("%s: %w", loc.Label, result.Error))
		}
	}

	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}
	return results, nil
}

func (n *Notifier) processLocation(loc config.Location, tomorrow time.Time, from, to string, dryRun bool) NotificationResult {
	result := NotificationResult{Label: loc.Label}

	s, err := n.ScraperFactory.NewScraper(loc.Scraper)
	if err != nil {
		result.Error = err
		return result
	}

	log.Printf("Scraping bin times for %s (%s - %s)", loc.Label, loc.AddressCode, loc.PostCode)
	binTimes, err := s.ScrapeBinTimes(loc.PostCode, loc.AddressCode)
	if err != nil {
		result.Error = err
		return result
	}

	for _, binTime := range binTimes {
		log.Printf("[%s] Next collection for %s is %s", loc.Label, binTime.Type, binTime.CollectionTime.String())
		if dateutil.IsDateMatching(binTime.CollectionTime, tomorrow) {
			result.Collections = append(result.Collections, binTime.Type)
		}
	}

	if len(result.Collections) > 0 {
		result.Message = fmt.Sprintf("%s: %s collection tomorrow", loc.Label, strings.Join(result.Collections, ", "))
		log.Printf("[%s] %s", loc.Label, result.Message)
		if err := n.SMSClient.SendSms(from, to, result.Message, dryRun); err != nil {
			result.Error = err
			return result
		}
		result.SMSSent = true
	} else if tomorrow.Weekday() == loc.CollectionDay {
		result.Message = fmt.Sprintf("%s: No collections scheduled for tomorrow (regular collection day)", loc.Label)
		log.Printf("[%s] %s", loc.Label, result.Message)
		if err := n.SMSClient.SendSms(from, to, result.Message, dryRun); err != nil {
			result.Error = err
			return result
		}
		result.SMSSent = true
	} else {
		log.Printf("[%s] No collections tomorrow, not a regular collection day", loc.Label)
	}

	return result
}

type twilioSMSClientAdapter struct {
	client *clients.TwilioClient
}

func (a *twilioSMSClientAdapter) SendSms(from string, to string, body string, dryRun bool) error {
	_, err := a.client.SendSms(from, to, body, dryRun)
	return err
}

func main() {
	f, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadConfig(f.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	cfg.DryRun = f.DryRun
	cfg.TodayDate = f.TodayDate

	notifier := &Notifier{
		ScraperFactory: &defaultScraperFactory{},
		SMSClient:      &twilioSMSClientAdapter{client: clients.NewTwilioClient()},
		Clock:          time.Now,
	}

	_, err = notifier.Run(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./cmd/notifier/ -v`
Expected: ALL PASS

**Step 5: Run all tests**

Run: `go test ./...`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add cmd/notifier/main.go cmd/notifier/main_test.go
git commit -m "feat: refactor notifier to support multiple locations with per-location SMS"
```

---

### Task 7: Add gopkg.in/yaml.v3 as direct dependency

**Files:**
- Modify: `go.mod`

**Step 1: Tidy the module**

Run: `go mod tidy && go mod vendor`

This will promote `gopkg.in/yaml.v3` from indirect to direct since `pkg/config/config.go` now imports it.

**Step 2: Verify tests still pass**

Run: `go test ./...`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add go.mod go.sum vendor/
git commit -m "chore: add gopkg.in/yaml.v3 as direct dependency"
```

---

### Task 8: Update Dockerfile

**Files:**
- Modify: `Dockerfile`

**Step 1: Update Dockerfile**

No test needed — this is a configuration change. Update the Dockerfile to remove the old ENTRYPOINT arguments and document the config file mount:

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy module files and vendor directory
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin-notifier ./cmd/notifier

# Runtime stage with headless Chrome
FROM chromedp/headless-shell:stable

# Install ca-certificates for HTTPS requests (Twilio API)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/bin-notifier /usr/local/bin/bin-notifier

# Create non-root user for security
RUN useradd -r -u 1001 appuser
USER appuser

ENTRYPOINT ["/usr/local/bin/bin-notifier"]
```

Note: The Dockerfile itself doesn't actually change — the ENTRYPOINT is the same. The difference is in how you invoke the Docker container (mount the config file). This is documented in the README update.

**Step 2: Commit (skip if no actual changes)**

If unchanged, skip this commit.

---

### Task 9: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Rewrite README**

Update the README to reflect the new YAML config file approach, new CLI flags, multi-location support, and Docker usage with config file mount. Key sections to update:

- Features: mention multi-location and pluggable scrapers
- Configuration: replace CLI flag tables with YAML config file docs
- Usage examples: show config.yaml, new CLI invocation, Docker with -v mount
- Architecture: update directory tree to show bracknell.go, wokingham.go
- Remove go-flags from key dependencies, add gopkg.in/yaml.v3

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for multi-location YAML config"
```

---

### Task 10: Remove go-flags dependency (if no longer needed)

**Files:**
- Modify: `go.mod`, `go.sum`, `vendor/`

**Step 1: Check if go-flags is still used**

go-flags is still used in `pkg/config/config.go` for the `Flags` struct (`ParseFlags`). So it stays. No action needed.

Skip this task.

---

### Task 11: Verify TODOS.md and final check

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 2: Build the binary**

Run: `go build -o bin-notifier ./cmd/notifier`
Expected: Builds without errors

**Step 3: Check for any TODOS.md to update**

If a TODOS.md exists, verify all sub-tasks are checked off.

**Step 4: Final commit if needed**

---

## Task Dependency Order

```
Task 1 (dateutil) → Task 2 (config) → Task 3 (flags)
Task 4 (scraper refactor) → Task 5 (wokingham stub)
Task 1-5 → Task 6 (notifier refactor)
Task 6 → Task 7 (go mod tidy)
Task 7 → Task 8 (Dockerfile) → Task 9 (README)
Task 9 → Task 11 (final verification)
```

Tasks 1-3 and Tasks 4-5 can be done in parallel.
