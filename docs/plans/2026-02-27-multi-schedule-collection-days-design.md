# Multi-Schedule Collection Days

## Problem

Each location currently supports a single `collection_day` field. In reality, different refuse types are collected on different days with different frequencies (e.g., recycling weekly on Tuesday, garden waste fortnightly on Friday). The application needs to support multiple collection schedules per location.

## Design Decisions

- **Schedules work alongside scraping.** The scraper remains the source of truth for actual collection dates. Configured schedules replace the single `collection_day` and provide "expected collection day" warnings when scraping returns no results for an expected day.
- **Frequency support:** `every_n_weeks` (defaults to 1 for weekly). Supports arbitrary intervals.
- **Fixed reference date** for non-weekly schedules. A `reference_date` field specifies any known collection date. The app calculates whether the current week is an "on" week using modular arithmetic. The config never needs updating.
- **Per-schedule warnings.** When tomorrow is an expected collection day but nothing is scraped, the warning names the specific refuse types expected.
- **Replaces `collection_day` entirely.** No backwards compatibility — all locations must use `collection_days`.

## Config Structure

```yaml
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
```

### Go Structs

```go
type CollectionDay struct {
    Day           time.Weekday `yaml:"-"`
    RawDay        string       `yaml:"day"`
    Types         []string     `yaml:"types"`
    EveryNWeeks   int          `yaml:"every_n_weeks"`  // defaults to 1
    ReferenceDate string       `yaml:"reference_date"` // required when every_n_weeks > 1
}

type Location struct {
    Label          string          `yaml:"label"`
    Scraper        string          `yaml:"scraper"`
    PostCode       string          `yaml:"postcode"`
    AddressCode    string          `yaml:"address_code"`
    CollectionDays []CollectionDay `yaml:"collection_days"`
}
```

### Validation Rules

- `collection_days` must have at least one entry.
- Each entry must have `day` (valid weekday) and at least one entry in `types`.
- `every_n_weeks` defaults to 1 if omitted; must be >= 1.
- `reference_date` required when `every_n_weeks > 1`; must parse as `YYYY-MM-DD` and fall on the same weekday as `day`.

## Matching Logic

In `processLocation()`:

1. Scrape bin times as before.
2. Check scraped results against tomorrow — collect matching `BinTime` entries.
3. If collections found tomorrow, send SMS listing them (unchanged).
4. If no collections found tomorrow, check each `CollectionDay` entry:
   - Is tomorrow the right weekday?
   - If `every_n_weeks > 1`, is this an "on" week? Calculate `weeks_elapsed = days_between(reference_date, tomorrow) / 7`, check `weeks_elapsed % every_n_weeks == 0`.
   - If matched, send per-schedule warning naming the expected types.
5. Multiple warnings possible if tomorrow matches multiple `CollectionDay` entries.

### `IsOnWeek` function (in `pkg/dateutil/`)

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

## SMS Message Format

**Collections found:**
```
Home: Tomorrows bin collections are: Recycling, General Waste
```

**Expected day, nothing scraped:**
```
Home: Expected Recycling, General Waste collection tomorrow (Tuesday) but none scheduled.
```

**Non-matching day or off-week:** No message.

## Files Changed

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `CollectionDay` struct. Replace fields on `Location`. Update `validate()`. |
| `pkg/config/config_test.go` | Update tests for new config structure. |
| `pkg/dateutil/dateutil.go` | Add `IsOnWeek()` function. |
| `pkg/dateutil/dateutil_test.go` | Add tests for `IsOnWeek()`. |
| `cmd/notifier/main.go` | Update `processLocation()` to loop over `CollectionDays`. |
| `cmd/notifier/main_test.go` | Update all tests, add multi-day/fortnightly/per-schedule warning tests. |
| `README.md` | Update config example and location fields table. |
| `CLAUDE.md` | Update config structure documentation. |

### Unchanged

- `pkg/scraper/` — scraper interface and implementations unaffected.
- `pkg/clients/` — SMS client unaffected.
- `pkg/regexp/` — unchanged.
