# Multi-Schedule Collection Days Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the single `collection_day` field per location with a `collection_days` list supporting multiple refuse types, days, and frequencies (weekly/every-N-weeks).

**Architecture:** Add a `CollectionDay` struct to config with day, types, frequency, and reference date. Add `IsOnWeek()` to dateutil for frequency calculation. Update `processLocation()` to loop over collection days for per-schedule "expected but not found" warnings.

**Tech Stack:** Go, stdlib `time`, `yaml.v3`, `testify`

---

### Task 1: Add `IsOnWeek` to dateutil (TDD)

**Files:**
- Modify: `pkg/dateutil/dateutil.go`
- Modify: `pkg/dateutil/dateutil_test.go`

**Step 1: Write the failing tests**

Add to `pkg/dateutil/dateutil_test.go`:

```go
func TestIsOnWeek(t *testing.T) {
	ref := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC) // a Friday

	tests := []struct {
		name        string
		reference   time.Time
		target      time.Time
		everyNWeeks int
		expected    bool
	}{
		{
			name:        "same week is on",
			reference:   ref,
			target:      ref,
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "1 week later is off for fortnightly",
			reference:   ref,
			target:      ref.AddDate(0, 0, 7),
			everyNWeeks: 2,
			expected:    false,
		},
		{
			name:        "2 weeks later is on for fortnightly",
			reference:   ref,
			target:      ref.AddDate(0, 0, 14),
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "3 weeks later is off for fortnightly",
			reference:   ref,
			target:      ref.AddDate(0, 0, 21),
			everyNWeeks: 2,
			expected:    false,
		},
		{
			name:        "every 3 weeks - week 3 is on",
			reference:   ref,
			target:      ref.AddDate(0, 0, 21),
			everyNWeeks: 3,
			expected:    true,
		},
		{
			name:        "every 3 weeks - week 2 is off",
			reference:   ref,
			target:      ref.AddDate(0, 0, 14),
			everyNWeeks: 3,
			expected:    false,
		},
		{
			name:        "weekly is always on",
			reference:   ref,
			target:      ref.AddDate(0, 0, 7),
			everyNWeeks: 1,
			expected:    true,
		},
		{
			name:        "reference date in future still works",
			reference:   ref.AddDate(0, 0, 14),
			target:      ref,
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "reference date in future - off week",
			reference:   ref.AddDate(0, 0, 14),
			target:      ref.AddDate(0, 0, 7),
			everyNWeeks: 2,
			expected:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsOnWeek(test.reference, test.target, test.everyNWeeks)
			assert.Equal(t, test.expected, actual)
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestIsOnWeek ./pkg/dateutil/`
Expected: FAIL — `IsOnWeek` not defined

**Step 3: Write the implementation**

Add to `pkg/dateutil/dateutil.go`:

```go
func IsOnWeek(referenceDate, targetDate time.Time, everyNWeeks int) bool {
	days := int(targetDate.Sub(referenceDate).Hours() / 24)
	if days < 0 {
		days = -days
	}
	weeks := days / 7
	return weeks%everyNWeeks == 0
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -run TestIsOnWeek ./pkg/dateutil/`
Expected: PASS

**Step 5: Run full dateutil tests to check no regressions**

Run: `go test ./pkg/dateutil/`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/dateutil/dateutil.go pkg/dateutil/dateutil_test.go
git commit -m "feat: add IsOnWeek function for multi-week collection frequency"
```

---

### Task 2: Update config structs and validation (TDD)

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

**Step 1: Update config tests**

Replace the entire test file `pkg/config/config_test.go`. Key changes:
- All YAML snippets change from `collection_day: tuesday` to `collection_days:` list format
- New tests for: `every_n_weeks` defaulting, `reference_date` validation, missing `types`, weekday mismatch on reference_date

```go
// In TestLoadConfig_ValidMultipleLocations, change YAML to:
//   collection_days:
//     - day: tuesday
//       types: ["Recycling", "General Waste"]
// And assertions from cfg.Locations[0].CollectionDay to:
//   cfg.Locations[0].CollectionDays[0].Day == time.Tuesday
//   cfg.Locations[0].CollectionDays[0].Types == ["Recycling", "General Waste"]
//   cfg.Locations[0].CollectionDays[0].EveryNWeeks == 1  (defaulted)

// In TestLoadConfig_CaseInsensitiveDay, change YAML to:
//   collection_days:
//     - day: WEDNESDAY
//       types: ["General Waste"]
// Assert cfg.Locations[0].CollectionDays[0].Day == time.Wednesday

// In TestLoadConfig_InvalidDay, change YAML to:
//   collection_days:
//     - day: notaday
//       types: ["General Waste"]

// In TestLoadConfig_MissingRequiredFields, replace "missing collection_day" case with:
//   "missing collection_days" — omit collection_days entirely, expect "collection_days is required"

// Add new test cases to TestLoadConfig_MissingRequiredFields:
//   "empty collection_days" — collection_days: [], expect "collection_days must have at least one entry"
//   "missing types" — collection_days with day but no types, expect "types is required"
//   "empty types" — types: [], expect "types must have at least one entry"
//   "every_n_weeks less than 1" — every_n_weeks: 0, expect "every_n_weeks must be >= 1"
//   "missing reference_date for fortnightly" — every_n_weeks: 2 without reference_date, expect "reference_date is required"
//   "invalid reference_date format" — reference_date: "not-a-date", expect "invalid reference_date"
//   "reference_date weekday mismatch" — day: tuesday, reference_date on a Wednesday, expect "reference_date must fall on"
```

Add a new test:

```go
func TestLoadConfig_FortnightlySchedule(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling", "General Waste"]
      - day: friday
        every_n_weeks: 2
        reference_date: "2026-01-02"
        types: ["Garden Waste"]
`)

	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Len(t, cfg.Locations[0].CollectionDays, 2)

	cd0 := cfg.Locations[0].CollectionDays[0]
	assert.Equal(t, time.Tuesday, cd0.Day)
	assert.Equal(t, []string{"Recycling", "General Waste"}, cd0.Types)
	assert.Equal(t, 1, cd0.EveryNWeeks)
	assert.Equal(t, "", cd0.ReferenceDate)

	cd1 := cfg.Locations[0].CollectionDays[1]
	assert.Equal(t, time.Friday, cd1.Day)
	assert.Equal(t, []string{"Garden Waste"}, cd1.Types)
	assert.Equal(t, 2, cd1.EveryNWeeks)
	assert.Equal(t, "2026-01-02", cd1.ReferenceDate)
}

func TestLoadConfig_EveryNWeeksDefaultsTo1(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling"]
`)

	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, 1, cfg.Locations[0].CollectionDays[0].EveryNWeeks)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/config/`
Expected: FAIL — `CollectionDays` field doesn't exist yet

**Step 3: Update config.go structs and validation**

In `pkg/config/config.go`:

Replace the `Location` struct:

```go
type CollectionDay struct {
	Day           time.Weekday `yaml:"-"`
	RawDay        string       `yaml:"day"`
	Types         []string     `yaml:"types"`
	EveryNWeeks   int          `yaml:"every_n_weeks"`
	ReferenceDate string       `yaml:"reference_date"`
}

type Location struct {
	Label          string          `yaml:"label"`
	Scraper        string          `yaml:"scraper"`
	PostCode       string          `yaml:"postcode"`
	AddressCode    string          `yaml:"address_code"`
	CollectionDays []CollectionDay `yaml:"collection_days"`
}
```

Remove the old `CollectionDay time.Weekday` and `RawDay string` fields from `Location`.

Update `validate()` — replace the `collection_day` validation block with:

```go
if len(loc.CollectionDays) == 0 {
    return fmt.Errorf("location %d: collection_days must have at least one entry", i+1)
}
for j := range loc.CollectionDays {
    cd := &loc.CollectionDays[j]
    if cd.RawDay == "" {
        return fmt.Errorf("location %d, schedule %d: day is required", i+1, j+1)
    }
    day, err := dateutil.ParseWeekday(cd.RawDay)
    if err != nil {
        return fmt.Errorf("location %d, schedule %d: %w", i+1, j+1, err)
    }
    cd.Day = day
    if len(cd.Types) == 0 {
        return fmt.Errorf("location %d, schedule %d: types must have at least one entry", i+1, j+1)
    }
    if cd.EveryNWeeks == 0 {
        cd.EveryNWeeks = 1
    }
    if cd.EveryNWeeks < 1 {
        return fmt.Errorf("location %d, schedule %d: every_n_weeks must be >= 1", i+1, j+1)
    }
    if cd.EveryNWeeks > 1 {
        if cd.ReferenceDate == "" {
            return fmt.Errorf("location %d, schedule %d: reference_date is required when every_n_weeks > 1", i+1, j+1)
        }
        refDate, err := time.Parse("2006-01-02", cd.ReferenceDate)
        if err != nil {
            return fmt.Errorf("location %d, schedule %d: invalid reference_date: %w", i+1, j+1, err)
        }
        if refDate.Weekday() != day {
            return fmt.Errorf("location %d, schedule %d: reference_date must fall on %s", i+1, j+1, day)
        }
    }
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/config/`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: replace collection_day with collection_days supporting multiple schedules"
```

---

### Task 3: Update processLocation and notifier tests (TDD)

**Files:**
- Modify: `cmd/notifier/main.go`
- Modify: `cmd/notifier/main_test.go`

**Step 1: Update existing test helpers and tests**

In `cmd/notifier/main_test.go`:

Update `createTestConfig()`:

```go
func createTestConfig() config.Config {
	return config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:         time.Tuesday,
						Types:       []string{"General Waste", "Recycling"},
						EveryNWeeks: 1,
					},
				},
			},
		},
	}
}
```

Update `TestNotifier_SendsSmsOnRegularDayNoCollections`:
- Replace `cfg.Locations[0].CollectionDay = time.Tuesday` with the `CollectionDays` slice already set via `createTestConfig()`
- Update assertion: message should now say `"Expected General Waste, Recycling collection tomorrow (Tuesday) but none scheduled."` instead of the generic "regular bin collection day" message

Update `TestNotifier_NoSmsWhenNoCollectionsAndNotRegularDay`:
- Replace `cfg.Locations[0].CollectionDay = time.Wednesday` with setting `CollectionDays` to have `Day: time.Wednesday`

Update `TestNotifier_ScraperErrorContinuesOtherLocations` and `TestNotifier_MultipleLocations`:
- Replace `CollectionDay: time.Tuesday` with `CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}`

Update `TestNotifier_UnknownScraperRecordsError`:
- Same replacement pattern

Add new tests:

```go
func TestNotifier_FortnightlyOnWeekSendsWarning(t *testing.T) {
	// Reference date is a Friday. Target tomorrow is 2 weeks later (on week).
	refDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) // Friday
	today := refDate.AddDate(0, 0, 13)                       // Thursday, 2 weeks - 1 day later
	// tomorrow is Friday, 2 weeks after reference = on week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{}, // nothing scraped
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:           time.Friday,
						Types:         []string{"Garden Waste"},
						EveryNWeeks:   2,
						ReferenceDate: "2026-01-02",
					},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.True(t, results[0].SMSSent)
	assert.Contains(t, results[0].Message, "Expected Garden Waste collection tomorrow (Friday)")
}

func TestNotifier_FortnightlyOffWeekNoMessage(t *testing.T) {
	refDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) // Friday
	today := refDate.AddDate(0, 0, 6)                        // Thursday, 1 week - 1 day later
	// tomorrow is Friday, 1 week after reference = off week for fortnightly

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:           time.Friday,
						Types:         []string{"Garden Waste"},
						EveryNWeeks:   2,
						ReferenceDate: "2026-01-02",
					},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.False(t, results[0].SMSSent)
	assert.Len(t, mockSMS.calls, 0)
}

func TestNotifier_MultipleCollectionDaysWarnings(t *testing.T) {
	// Tomorrow is Tuesday. Two schedules match Tuesday.
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek}, // not tomorrow
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
					{Day: time.Tuesday, Types: []string{"Food Waste"}, EveryNWeeks: 1},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.True(t, results[0].SMSSent)
	// Should have sent 2 warning SMS (one per schedule)
	assert.Len(t, mockSMS.calls, 2)
	assert.Contains(t, mockSMS.calls[0].body, "Recycling")
	assert.Contains(t, mockSMS.calls[1].body, "Food Waste")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/notifier/`
Expected: FAIL — `processLocation` still uses old `CollectionDay` field

**Step 3: Update processLocation in main.go**

Replace the `else if` and final `else` blocks in `processLocation()` (lines 100-112) with:

```go
	if len(result.Collections) != 0 {
		result.Message = loc.Label + ": Tomorrows bin collections are: " + strings.Join(result.Collections, ", ")
		log.Printf("[%s] %s", loc.Label, result.Message)

		err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, result.Message, cfg.DryRun)
		if err != nil {
			result.Error = fmt.Errorf("[%s] SMS error: %w", loc.Label, err)
			return result
		}
		result.SMSSent = true
	} else {
		for _, cd := range loc.CollectionDays {
			if tomorrow.Weekday() != cd.Day {
				continue
			}
			if cd.EveryNWeeks > 1 {
				refDate, _ := time.Parse("2006-01-02", cd.ReferenceDate)
				if !dateutil.IsOnWeek(refDate, tomorrow, cd.EveryNWeeks) {
					continue
				}
			}
			msg := fmt.Sprintf("%s: Expected %s collection tomorrow (%s) but none scheduled.",
				loc.Label, strings.Join(cd.Types, ", "), tomorrow.Weekday())
			log.Printf("[%s] %s", loc.Label, msg)
			result.Message = msg

			err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, msg, cfg.DryRun)
			if err != nil {
				result.Error = fmt.Errorf("[%s] SMS error: %w", loc.Label, err)
				return result
			}
			result.SMSSent = true
		}
		if !result.SMSSent {
			log.Printf("[%s] No collections tomorrow and not an expected collection day", loc.Label)
		}
	}
```

Add `"fmt"` to imports if not already present (it is).

**Step 4: Run tests to verify they pass**

Run: `go test ./cmd/notifier/`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/notifier/main.go cmd/notifier/main_test.go
git commit -m "feat: update processLocation for multi-schedule collection day warnings"
```

---

### Task 4: Update README and CLAUDE.md

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Update README.md config example**

Replace the YAML config example (around line 54-70) with:

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
    scraper: "bracknell"
    postcode: "RG12 9ZZ"
    address_code: "654321"
    collection_days:
      - day: "Friday"
        types: ["General Waste", "Recycling"]
```

Replace the location fields table (around line 74-81) with:

```markdown
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
```

**Step 2: Update CLAUDE.md config structure section**

Update the `Config Structure` subsection under Architecture to reflect:
- `Location` now has `CollectionDays []CollectionDay` instead of `CollectionDay time.Weekday`
- Add `CollectionDay` struct description

**Step 3: Verify the build still works**

Run: `go build -o bin-notifier ./cmd/notifier`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: update README and CLAUDE.md for multi-schedule collection days"
```

---

### Task 5: Final verification

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: All tests PASS

**Step 2: Build the binary**

Run: `go build -o bin-notifier ./cmd/notifier`
Expected: Builds successfully

**Step 3: Clean up build artifact**

Run: `rm -f bin-notifier`
