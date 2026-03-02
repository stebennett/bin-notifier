# Wokingham Scraper Design

## Overview

Implement the Wokingham Borough Council bin collection scraper, replacing the current stub with a working chromedp-based implementation that follows the same pattern as the Bracknell scraper.

## Page Flow

Target URL: `https://www.wokingham.gov.uk/rubbish-and-recycling/waste-collection/find-your-bin-collection-day`

1. Navigate to page, accept cookie banner (`.agree-button`)
2. Enter postcode in `#edit-postcode-search-csv`, click "Find Address" (`#edit-find-address`)
3. Wait for address dropdown (`#edit-address-options-csv`), select by UPRN value (the `addressCode` config field)
4. Click "Show collection dates" (`#edit-show-collection-dates-csv`)
5. Wait for results cards (`.cards-list`)

## Results Structure

Results are server-rendered Drupal cards inside `div.cards-list`. Each card:

```html
<div class="card card--waste card--blue-light">
  <div class="card__wrapper">
    <div class="card__content">
      <h3 class="heading heading--sub heading--tiny">
        Household waste (week 2)
      </h3>
      <p class="paragraph">Your next collection will be:</p>
      <span class="card__date">
        Today 27/02/2026
      </span>
    </div>
  </div>
</div>
```

Four collection types observed:
- Household waste
- Garden waste
- Recycling
- Food waste

## Parsing

- **Type:** Extract from `h3` text, strip week info with regex: `(?P<BinType>[A-Za-z\s]+?)(?:\s*\(week \d+\))?$`
- **Date:** Extract DD/MM/YYYY from `span.card__date` text with regex: `(?P<Day>\d{2})/(?P<Month>\d{2})/(?P<Year>\d{4})`
- Convert to `time.Time` using `dateutil.AsTime(day, month, year)`

## Config

The `addressCode` field stores the UPRN value from the address dropdown (e.g. `"120033"` for 61 The Brambles, Crowthorne).

Example config:
```yaml
locations:
  - label: "Home"
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

## Error Handling

- Validate postcode and addressCode are non-empty before launching Chrome
- 60-second timeout on chromedp context
- Return error if no collection cards found after form submission
- Return error if date parsing fails for any card

## Testing

- Unit tests for `parseWokinghamCollection()` helper (type + date extraction)
- Error cases: empty input, malformed dates, missing fields
- Factory test: `NewScraper("wokingham")` returns `*WokinghamScraper`

## Implementation Approach

- Modify `pkg/scraper/wokingham.go` to replace the stub
- Add `parseWokinghamCollection()` as an exported function for testability
- Add tests in `pkg/scraper/scraper_test.go`
