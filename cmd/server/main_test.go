package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stebennett/bin-notifier/pkg/cache"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScraper struct {
	binTimes []scraper.BinTime
	err      error
}

func (m *mockScraper) ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error) {
	return m.binTimes, m.err
}

func newMockFactory(scrapers map[string]*mockScraper) ScraperFactory {
	return func(name string) (BinScraper, error) {
		s, ok := scrapers[name]
		if !ok {
			return nil, errors.New("unknown scraper: " + name)
		}
		return s, nil
	}
}

func testApp(locations []config.Location, scrapers map[string]*mockScraper, now time.Time) *App {
	return &App{
		cfg: config.Config{
			Locations: locations,
		},
		scraperFactory: newMockFactory(scrapers),
		cache:          cache.New(6 * time.Hour),
		now:            func() time.Time { return now },
	}
}

func callTool(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func testLocations() []config.Location {
	return []config.Location{
		{
			Label:       "Home",
			Scraper:     "bracknell",
			PostCode:    "RG12 1AB",
			AddressCode: "12345",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling", "General Waste"}, EveryNWeeks: 1},
				{
					Day:           time.Friday,
					Types:         []string{"Garden Waste"},
					EveryNWeeks:   2,
					ReferenceDate: "2026-01-02",
				},
			},
		},
		{
			Label:       "Office",
			Scraper:     "wokingham",
			PostCode:    "RG42 2XY",
			AddressCode: "67890",
			CollectionDays: []config.CollectionDay{
				{Day: time.Thursday, Types: []string{"General Waste"}, EveryNWeeks: 1},
			},
		},
	}
}

// --- get_collections tests ---

func TestGetCollections_Tomorrow(t *testing.T) {
	// Monday 2026-03-16, tomorrow is Tuesday
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Equal(t, "schedule", resp.Source)
	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "2026-03-17", resp.Collections[0].Date)
	assert.Equal(t, "Home", resp.Collections[0].Location)
	assert.Equal(t, []string{"Recycling", "General Waste"}, resp.Collections[0].Types)
}

func TestGetCollections_Today(t *testing.T) {
	// Tuesday 2026-03-17
	now := time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{"range": "today"}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "2026-03-17", resp.Collections[0].Date)
}

func TestGetCollections_ThisWeek(t *testing.T) {
	// Monday 2026-03-16
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{"range": "this_week"}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	// Tuesday: Home (Recycling, General Waste), Thursday: Office (General Waste), Friday: Home (Garden Waste — check if on week)
	// 2026-03-20 is Friday, reference 2026-01-02. Weeks diff = (2026-03-20 - 2026-01-02) / 7 = 77/7 = 11 weeks. 11 % 2 = 1, so OFF week.
	assert.Len(t, resp.Collections, 2)
	assert.Equal(t, "Home", resp.Collections[0].Location)
	assert.Equal(t, "Office", resp.Collections[1].Location)
}

func TestGetCollections_SpecificDate(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{"date": "2026-03-19"}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	// 2026-03-19 is Thursday
	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "Office", resp.Collections[0].Location)
}

func TestGetCollections_LocationFilter(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{"range": "this_week", "location": "office"}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "Office", resp.Collections[0].Location)
}

func TestGetCollections_InvalidDate(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{"date": "bad-date"}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetCollections_NoResults(t *testing.T) {
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC) // Sunday
	app := testApp(testLocations(), nil, now)

	// Tomorrow is Monday — no collections configured for Monday
	result, err := app.handleGetCollections(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)

	var resp collectionsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))
	assert.Empty(t, resp.Collections)
}

// --- get_next_collection tests ---

func TestGetNextCollection_ReturnsScrapedData(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	scrapers := map[string]*mockScraper{
		"bracknell": {
			binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
				{Type: "General Waste", CollectionTime: tomorrow},
			},
		},
		"wokingham": {
			binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow.AddDate(0, 0, 2)},
			},
		},
	}

	app := testApp(testLocations(), scrapers, now)

	result, err := app.handleGetNextCollection(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)

	var resp nextCollectionResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Equal(t, "scraper", resp.Source)
	assert.Len(t, resp.Collections, 3)
}

func TestGetNextCollection_FilterByBinType(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	scrapers := map[string]*mockScraper{
		"bracknell": {
			binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
				{Type: "General Waste", CollectionTime: tomorrow},
			},
		},
		"wokingham": {
			binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow},
			},
		},
	}

	app := testApp(testLocations(), scrapers, now)

	result, err := app.handleGetNextCollection(context.Background(), callTool(map[string]any{"bin_type": "recycling"}))
	require.NoError(t, err)

	var resp nextCollectionResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "Recycling", resp.Collections[0].Type)
	assert.Equal(t, "Home", resp.Collections[0].Location)
}

func TestGetNextCollection_FilterByLocation(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	scrapers := map[string]*mockScraper{
		"wokingham": {
			binTimes: []scraper.BinTime{
				{Type: "General Waste", CollectionTime: tomorrow},
			},
		},
	}

	app := testApp(testLocations(), scrapers, now)

	result, err := app.handleGetNextCollection(context.Background(), callTool(map[string]any{"location": "Office"}))
	require.NoError(t, err)

	var resp nextCollectionResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Len(t, resp.Collections, 1)
	assert.Equal(t, "Office", resp.Collections[0].Location)
}

func TestGetNextCollection_CachesResults(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	callCount := 0
	scrapers := map[string]*mockScraper{
		"bracknell": {
			binTimes: []scraper.BinTime{
				{Type: "Recycling", CollectionTime: tomorrow},
			},
		},
	}

	locs := []config.Location{testLocations()[0]} // Home only
	app := testApp(locs, scrapers, now)

	// Wrap factory to count calls
	originalFactory := app.scraperFactory
	app.scraperFactory = func(name string) (BinScraper, error) {
		callCount++
		return originalFactory(name)
	}

	// First call should scrape
	_, err := app.handleGetNextCollection(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	_, err = app.handleGetNextCollection(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestGetNextCollection_ScraperError(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)

	scrapers := map[string]*mockScraper{
		"bracknell": {err: errors.New("chrome not found")},
	}

	locs := []config.Location{testLocations()[0]}
	app := testApp(locs, scrapers, now)

	result, err := app.handleGetNextCollection(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- list_locations tests ---

func TestListLocations(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	app := testApp(testLocations(), nil, now)

	result, err := app.handleListLocations(context.Background(), callTool(map[string]any{}))
	require.NoError(t, err)

	var resp listLocationsResponse
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp))

	assert.Len(t, resp.Locations, 2)
	assert.Equal(t, "Home", resp.Locations[0].Label)
	assert.Equal(t, "bracknell", resp.Locations[0].Scraper)
	assert.Equal(t, "RG12 1AB", resp.Locations[0].PostCode)
	assert.Len(t, resp.Locations[0].CollectionDays, 2)
	assert.Equal(t, "Tuesday", resp.Locations[0].CollectionDays[0].Day)
	assert.Equal(t, "Friday", resp.Locations[0].CollectionDays[1].Day)
	assert.Equal(t, 2, resp.Locations[0].CollectionDays[1].EveryNWeeks)
	assert.Equal(t, "2026-01-02", resp.Locations[0].CollectionDays[1].ReferenceDate)

	assert.Equal(t, "Office", resp.Locations[1].Label)
	assert.Equal(t, "wokingham", resp.Locations[1].Scraper)
}

// --- resolveDateRange tests ---

func TestResolveDateRange_Today(t *testing.T) {
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC) // Friday
	from, to, err := resolveDateRange("today", "", today)
	assert.NoError(t, err)
	assert.Equal(t, today, from)
	assert.Equal(t, today, to)
}

func TestResolveDateRange_Tomorrow(t *testing.T) {
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	from, to, err := resolveDateRange("tomorrow", "", today)
	assert.NoError(t, err)
	expected := today.AddDate(0, 0, 1)
	assert.Equal(t, expected, from)
	assert.Equal(t, expected, to)
}

func TestResolveDateRange_ThisWeek(t *testing.T) {
	today := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC) // Wednesday
	from, to, err := resolveDateRange("this_week", "", today)
	assert.NoError(t, err)
	assert.Equal(t, today, from)
	assert.Equal(t, time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC), to) // Sunday
}

func TestResolveDateRange_ThisWeekFromSunday(t *testing.T) {
	today := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC) // Sunday
	from, to, err := resolveDateRange("this_week", "", today)
	assert.NoError(t, err)
	assert.Equal(t, today, from)
	assert.Equal(t, today, to)
}

func TestResolveDateRange_NextWeek(t *testing.T) {
	today := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC) // Wednesday
	from, to, err := resolveDateRange("next_week", "", today)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC), from) // Monday
	assert.Equal(t, time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), to)   // Sunday
}

func TestResolveDateRange_NextWeekFromMonday(t *testing.T) {
	today := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC) // Monday
	from, to, err := resolveDateRange("next_week", "", today)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC), from) // Next Monday
	assert.Equal(t, time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), to)   // Next Sunday
}

func TestResolveDateRange_SpecificDate(t *testing.T) {
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	from, to, err := resolveDateRange("", "2026-04-01", today)
	assert.NoError(t, err)
	expected := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, from)
	assert.Equal(t, expected, to)
}

func TestResolveDateRange_DefaultTomorrow(t *testing.T) {
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	from, to, err := resolveDateRange("", "", today)
	assert.NoError(t, err)
	expected := today.AddDate(0, 0, 1)
	assert.Equal(t, expected, from)
	assert.Equal(t, expected, to)
}

func TestResolveDateRange_RangeOverridesDate(t *testing.T) {
	today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	from, to, err := resolveDateRange("today", "2026-04-01", today)
	assert.NoError(t, err)
	assert.Equal(t, today, from)
	assert.Equal(t, today, to)
}
