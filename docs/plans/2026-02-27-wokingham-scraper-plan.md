# Wokingham Scraper Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Wokingham scraper stub with a working chromedp-based implementation that scrapes bin collection dates from wokingham.gov.uk.

**Architecture:** Follows the same pattern as the Bracknell scraper — a `WokinghamScraper` struct implementing `BinScraper`, using chromedp to navigate a Drupal form (postcode → address select → results cards), with a separate `parseWokinghamCollection()` function for testable parsing.

**Tech Stack:** Go, chromedp, existing `dateutil` and `regexp` packages.

---

### Task 1: Add parsing tests

**Files:**
- Modify: `pkg/scraper/scraper_test.go`

**Step 1: Write failing tests for `parseWokinghamCollection`**

Add these tests after the existing `TestWokinghamScraper_ReturnsNotImplemented` test:

```go
func TestParseWokinghamCollection(t *testing.T) {
	tests := []struct {
		name         string
		heading      string
		dateText     string
		expectedType string
		expectedTime time.Time
	}{
		{
			name:         "household waste with week",
			heading:      "Household waste (week 2)",
			dateText:     "Today 27/02/2026",
			expectedType: "Household waste",
			expectedTime: dateutil.AsTime(27, 2, 2026),
		},
		{
			name:         "garden waste with week",
			heading:      "Garden waste (week 2)",
			dateText:     "Tuesday 10/03/2026",
			expectedType: "Garden waste",
			expectedTime: dateutil.AsTime(10, 3, 2026),
		},
		{
			name:         "recycling with week",
			heading:      "Recycling (week 1)",
			dateText:     "Friday 06/03/2026",
			expectedType: "Recycling",
			expectedTime: dateutil.AsTime(6, 3, 2026),
		},
		{
			name:         "food waste no week",
			heading:      "Food waste",
			dateText:     "Today 27/02/2026",
			expectedType: "Food waste",
			expectedTime: dateutil.AsTime(27, 2, 2026),
		},
		{
			name:         "type without week info",
			heading:      "Recycling",
			dateText:     "Monday 15/04/2026",
			expectedType: "Recycling",
			expectedTime: dateutil.AsTime(15, 4, 2026),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parseWokinghamCollection(test.heading, test.dateText)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedType, actual.Type)
			assert.Equal(t, test.expectedTime, actual.CollectionTime)
		})
	}
}
```

Note: you will need to add `"time"` to the test file imports.

**Step 2: Run tests to verify they fail**

Run: `go test -run TestParseWokinghamCollection ./pkg/scraper/ -v`
Expected: FAIL — `parseWokinghamCollection` is undefined.

---

### Task 2: Add parsing error tests

**Files:**
- Modify: `pkg/scraper/scraper_test.go`

**Step 1: Write failing error tests**

Add after the tests from Task 1:

```go
func TestParseWokinghamCollection_Errors(t *testing.T) {
	tests := []struct {
		name     string
		heading  string
		dateText string
	}{
		{
			name:     "empty heading",
			heading:  "",
			dateText: "Friday 06/03/2026",
		},
		{
			name:     "empty date",
			heading:  "Recycling (week 1)",
			dateText: "",
		},
		{
			name:     "malformed date - no slashes",
			heading:  "Recycling",
			dateText: "Friday 06-03-2026",
		},
		{
			name:     "malformed date - missing year",
			heading:  "Recycling",
			dateText: "Friday 06/03",
		},
		{
			name:     "random text as date",
			heading:  "Recycling",
			dateText: "No collection scheduled",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseWokinghamCollection(test.heading, test.dateText)
			assert.Error(t, err)
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestParseWokinghamCollection ./pkg/scraper/ -v`
Expected: FAIL — `parseWokinghamCollection` is undefined (same as Task 1).

---

### Task 3: Implement parsing function

**Files:**
- Modify: `pkg/scraper/wokingham.go`

**Step 1: Implement `parseWokinghamCollection`**

Replace the entire contents of `pkg/scraper/wokingham.go` with:

```go
package scraper

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
)

type WokinghamScraper struct{}

func parseWokinghamCollection(heading string, dateText string) (BinTime, error) {
	heading = strings.TrimSpace(heading)
	dateText = strings.TrimSpace(dateText)

	if len(heading) == 0 {
		return BinTime{}, errors.New("empty heading")
	}

	// Extract bin type, stripping "(week N)" suffix if present
	typeExp := regexp.MustCompile(`^(?P<BinType>[A-Za-z\s]+?)\s*(?:\(week \d+\))?$`)
	typeMatches := typeExp.FindStringSubmatch(heading)
	if typeMatches == nil {
		return BinTime{}, errors.New("failed to parse bin type from heading")
	}
	binType := strings.TrimSpace(typeMatches[1])

	// Extract DD/MM/YYYY from date text
	dateExp := regexp.MustCompile(`(?P<Day>\d{2})/(?P<Month>\d{2})/(?P<Year>\d{4})`)
	dateMatches := dateExp.FindStringSubmatch(dateText)
	if dateMatches == nil {
		return BinTime{}, errors.New("failed to parse date from date text")
	}

	day, _ := strconv.Atoi(dateMatches[1])
	month, _ := strconv.Atoi(dateMatches[2])
	year, _ := strconv.Atoi(dateMatches[3])

	return BinTime{
		Type:           binType,
		CollectionTime: dateutil.AsTime(day, month, year),
	}, nil
}

func (s *WokinghamScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	return nil, errors.New("wokingham scraper not implemented")
}
```

Note: Keep `ScrapeBinTimes` as a stub for now — the chromedp implementation comes in Task 5.

**Step 2: Run tests to verify they pass**

Run: `go test -run TestParseWokinghamCollection ./pkg/scraper/ -v`
Expected: All tests PASS.

**Step 3: Commit**

```bash
git add pkg/scraper/wokingham.go pkg/scraper/scraper_test.go
git commit -m "feat: add parseWokinghamCollection with tests"
```

---

### Task 4: Add validation tests for WokinghamScraper

**Files:**
- Modify: `pkg/scraper/scraper_test.go`

**Step 1: Write validation tests**

Replace the existing `TestWokinghamScraper_ReturnsNotImplemented` test with:

```go
func TestWokinghamScrapeBinTimes_ValidationErrors(t *testing.T) {
	scraper := &WokinghamScraper{}

	t.Run("empty postcode returns error", func(t *testing.T) {
		_, err := scraper.ScrapeBinTimes("", "123")
		assert.Error(t, err)
		assert.EqualError(t, err, "no postcode specified")
	})

	t.Run("empty address returns error", func(t *testing.T) {
		_, err := scraper.ScrapeBinTimes("AB1 2CD", "")
		assert.Error(t, err)
		assert.EqualError(t, err, "no address specified")
	})

	t.Run("both empty returns postcode error first", func(t *testing.T) {
		_, err := scraper.ScrapeBinTimes("", "")
		assert.Error(t, err)
		assert.EqualError(t, err, "no postcode specified")
	})
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestWokinghamScrapeBinTimes_ValidationErrors ./pkg/scraper/ -v`
Expected: FAIL — currently returns "not implemented" instead of "no postcode specified".

---

### Task 5: Implement WokinghamScraper.ScrapeBinTimes

**Files:**
- Modify: `pkg/scraper/wokingham.go`

**Step 1: Replace the stub `ScrapeBinTimes` with the full chromedp implementation**

Replace the `ScrapeBinTimes` method and add necessary imports. The full file should be:

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
	"github.com/stebennett/bin-notifier/pkg/dateutil"
)

type WokinghamScraper struct{}

func parseWokinghamCollection(heading string, dateText string) (BinTime, error) {
	heading = strings.TrimSpace(heading)
	dateText = strings.TrimSpace(dateText)

	if len(heading) == 0 {
		return BinTime{}, errors.New("empty heading")
	}

	// Extract bin type, stripping "(week N)" suffix if present
	typeExp := regexp.MustCompile(`^(?P<BinType>[A-Za-z\s]+?)\s*(?:\(week \d+\))?$`)
	typeMatches := typeExp.FindStringSubmatch(heading)
	if typeMatches == nil {
		return BinTime{}, errors.New("failed to parse bin type from heading")
	}
	binType := strings.TrimSpace(typeMatches[1])

	// Extract DD/MM/YYYY from date text
	dateExp := regexp.MustCompile(`(?P<Day>\d{2})/(?P<Month>\d{2})/(?P<Year>\d{4})`)
	dateMatches := dateExp.FindStringSubmatch(dateText)
	if dateMatches == nil {
		return BinTime{}, errors.New("failed to parse date from date text")
	}

	day, _ := strconv.Atoi(dateMatches[1])
	month, _ := strconv.Atoi(dateMatches[2])
	year, _ := strconv.Atoi(dateMatches[3])

	return BinTime{
		Type:           binType,
		CollectionTime: dateutil.AsTime(day, month, year),
	}, nil
}

func (s *WokinghamScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	if len(postCode) == 0 {
		return []BinTime{}, errors.New("no postcode specified")
	}
	if len(addressCode) == 0 {
		return []BinTime{}, errors.New("no address specified")
	}

	log.Printf("creating temp user data dir")
	dir, err := os.MkdirTemp("", "chromedp-wokingham")
	if err != nil {
		return []BinTime{}, err
	}
	defer os.RemoveAll(dir)

	log.Printf("setting chrome defaults")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", true),
		chromedp.NoSandbox,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	taskCtx, cancel = context.WithTimeout(taskCtx, 60*time.Second)
	defer cancel()

	log.Printf("navigating to wokingham waste collection page")

	// Step 1: Navigate and accept cookies
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("https://www.wokingham.gov.uk/rubbish-and-recycling/waste-collection/find-your-bin-collection-day"),
		chromedp.WaitVisible(`#edit-postcode-search-csv`, chromedp.ByQuery),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Accept cookies (ignore error if no banner)
	chromedp.Run(taskCtx,
		chromedp.Click(`.agree-button`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)

	// Step 2: Enter postcode and submit
	log.Printf("entering postcode: %s", postCode)
	err = chromedp.Run(taskCtx,
		chromedp.SetValue(`#edit-postcode-search-csv`, postCode, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`#edit-find-address`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Step 3: Select address and show collection dates
	log.Printf("selecting address: %s", addressCode)
	err = chromedp.Run(taskCtx,
		chromedp.WaitVisible(`#edit-address-options-csv`, chromedp.ByQuery),
		chromedp.SetValue(`#edit-address-options-csv`, addressCode, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`#edit-show-collection-dates-csv`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Step 4: Wait for results and extract card data
	log.Printf("extracting collection dates")
	var cardCount int
	err = chromedp.Run(taskCtx,
		chromedp.WaitVisible(`.cards-list`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll('.card--waste').length`, &cardCount),
	)
	if err != nil {
		return []BinTime{}, err
	}

	if cardCount == 0 {
		return []BinTime{}, errors.New("no collection cards found")
	}

	// Extract headings and dates from each card
	var headings, dates []string
	err = chromedp.Run(taskCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.card--waste h3')).map(h => h.textContent.trim())`, &headings),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.card--waste .card__date')).map(d => d.textContent.trim())`, &dates),
	)
	if err != nil {
		return []BinTime{}, err
	}

	if len(headings) != len(dates) {
		return []BinTime{}, errors.New("mismatched headings and dates count")
	}

	log.Printf("found %d collection cards, parsing", len(headings))

	binTimes := make([]BinTime, 0, len(headings))
	for i := range headings {
		bt, err := parseWokinghamCollection(headings[i], dates[i])
		if err != nil {
			return binTimes, err
		}
		binTimes = append(binTimes, bt)
	}

	return binTimes, nil
}
```

**Step 2: Run validation tests to verify they pass**

Run: `go test -run TestWokinghamScrapeBinTimes_ValidationErrors ./pkg/scraper/ -v`
Expected: All PASS.

**Step 3: Run all scraper tests**

Run: `go test ./pkg/scraper/ -v`
Expected: All PASS.

**Step 4: Commit**

```bash
git add pkg/scraper/wokingham.go pkg/scraper/scraper_test.go
git commit -m "feat: implement WokinghamScraper with chromedp"
```

---

### Task 6: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All PASS.

**Step 2: Build the binary**

Run: `go build -o bin-notifier ./cmd/notifier`
Expected: Builds successfully.

---

### Task 7: Update README

**Files:**
- Modify: `README.md`

**Step 1: Update the README to document the Wokingham scraper**

Find the section describing available scrapers and add Wokingham alongside Bracknell. Include:
- Scraper name: `wokingham`
- That `address_code` is the UPRN value from the council website's address dropdown
- Example config snippet showing a Wokingham location

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add Wokingham scraper to README"
```
