package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stebennett/bin-notifier/pkg/cache"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/schedule"
	"github.com/stebennett/bin-notifier/pkg/scraper"
)

// BinScraper is an interface for scraping bin collection times.
type BinScraper interface {
	ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error)
}

// ScraperFactory resolves a BinScraper by name.
type ScraperFactory func(name string) (BinScraper, error)

// App holds the shared state for MCP tool handlers.
type App struct {
	cfg            config.Config
	scraperFactory ScraperFactory
	cache          *cache.ScraperCache
	now            func() time.Time
}

func main() {
	configPath := os.Getenv("BN_CONFIG_FILE")
	for i, arg := range os.Args[1:] {
		if (arg == "-c" || arg == "--config") && i+1 < len(os.Args[1:]) {
			configPath = os.Args[i+2]
		}
	}
	if configPath == "" {
		log.Fatal("config file is required (-c or BN_CONFIG_FILE)")
	}

	cfg, err := config.LoadConfigForMCP(configPath)
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		cfg: cfg,
		scraperFactory: func(name string) (BinScraper, error) {
			return scraper.NewScraper(name)
		},
		cache: cache.New(6 * time.Hour),
		now:   time.Now,
	}

	s := server.NewMCPServer(
		"bin-notifier",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(getCollectionsTool(), app.handleGetCollections)
	s.AddTool(getNextCollectionTool(), app.handleGetNextCollection)
	s.AddTool(listLocationsTool(), app.handleListLocations)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// --- Tool definitions ---

func getCollectionsTool() mcp.Tool {
	return mcp.NewTool("get_collections",
		mcp.WithDescription("Get projected bin collections for a date or date range based on config schedule rules. Fast, no Chrome needed."),
		mcp.WithString("range",
			mcp.Description(`Date range: "today", "tomorrow", "this_week", "next_week". Default: "tomorrow"`),
			mcp.Enum("today", "tomorrow", "this_week", "next_week"),
		),
		mcp.WithString("date",
			mcp.Description("Specific date (YYYY-MM-DD). Overridden by range if both set."),
		),
		mcp.WithString("location",
			mcp.Description("Filter by location label (case-insensitive substring match)."),
		),
	)
}

func getNextCollectionTool() mcp.Tool {
	return mcp.NewTool("get_next_collection",
		mcp.WithDescription("Get the next confirmed collection date by scraping the council website. Results are cached for 6 hours."),
		mcp.WithString("bin_type",
			mcp.Description("Filter by bin type (case-insensitive substring match, e.g. \"recycling\")."),
		),
		mcp.WithString("location",
			mcp.Description("Filter by location label (case-insensitive substring match)."),
		),
	)
}

func listLocationsTool() mcp.Tool {
	return mcp.NewTool("list_locations",
		mcp.WithDescription("List all configured locations with their scrapers and collection day schedules."),
	)
}

// --- Tool handlers ---

type collectionsResponse struct {
	Collections []collectionEntry `json:"collections"`
	Source      string            `json:"source"`
}

type collectionEntry struct {
	Date     string   `json:"date"`
	Location string   `json:"location"`
	Types    []string `json:"types"`
}

func (a *App) handleGetCollections(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	now := a.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	rangeParam := request.GetString("range", "")
	dateParam := request.GetString("date", "")
	locationFilter := request.GetString("location", "")

	from, to, err := resolveDateRange(rangeParam, dateParam, today)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	locations := filterLocations(a.cfg.Locations, locationFilter)
	collections := schedule.ProjectCollections(locations, from, to)

	entries := make([]collectionEntry, len(collections))
	for i, c := range collections {
		entries[i] = collectionEntry{
			Date:     c.Date.Format("2006-01-02"),
			Location: c.Location,
			Types:    c.Types,
		}
	}

	resp := collectionsResponse{
		Collections: entries,
		Source:      "schedule",
	}
	return jsonResult(resp)
}

type nextCollectionResponse struct {
	Collections []nextCollectionEntry `json:"collections"`
	Source      string                `json:"source"`
}

type nextCollectionEntry struct {
	Location string   `json:"location"`
	Type     string   `json:"type"`
	Date     string   `json:"date"`
}

func (a *App) handleGetNextCollection(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	binTypeFilter := request.GetString("bin_type", "")
	locationFilter := request.GetString("location", "")

	locations := filterLocations(a.cfg.Locations, locationFilter)

	var entries []nextCollectionEntry
	var errs []string

	for _, loc := range locations {
		binTimes, ok := a.cache.Get(loc.PostCode, loc.AddressCode)
		if !ok {
			s, err := a.scraperFactory(loc.Scraper)
			if err != nil {
				errs = append(errs, fmt.Sprintf("[%s] scraper error: %v", loc.Label, err))
				continue
			}
			binTimes, err = s.ScrapeBinTimes(loc.PostCode, loc.AddressCode)
			if err != nil {
				errs = append(errs, fmt.Sprintf("[%s] scrape error: %v", loc.Label, err))
				continue
			}
			a.cache.Set(loc.PostCode, loc.AddressCode, binTimes)
		}

		for _, bt := range binTimes {
			if binTypeFilter != "" && !strings.Contains(strings.ToLower(bt.Type), strings.ToLower(binTypeFilter)) {
				continue
			}
			entries = append(entries, nextCollectionEntry{
				Location: loc.Label,
				Type:     bt.Type,
				Date:     bt.CollectionTime.Format("2006-01-02"),
			})
		}
	}

	if len(errs) > 0 && len(entries) == 0 {
		return mcp.NewToolResultError(strings.Join(errs, "; ")), nil
	}

	resp := nextCollectionResponse{
		Collections: entries,
		Source:      "scraper",
	}
	return jsonResult(resp)
}

type listLocationsResponse struct {
	Locations []locationInfo `json:"locations"`
}

type locationInfo struct {
	Label          string              `json:"label"`
	Scraper        string              `json:"scraper"`
	PostCode       string              `json:"postcode"`
	CollectionDays []collectionDayInfo `json:"collection_days"`
}

type collectionDayInfo struct {
	Day           string   `json:"day"`
	Types         []string `json:"types"`
	EveryNWeeks   int      `json:"every_n_weeks"`
	ReferenceDate string   `json:"reference_date,omitempty"`
}

func (a *App) handleListLocations(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var locs []locationInfo
	for _, loc := range a.cfg.Locations {
		var days []collectionDayInfo
		for _, cd := range loc.CollectionDays {
			days = append(days, collectionDayInfo{
				Day:           cd.Day.String(),
				Types:         cd.Types,
				EveryNWeeks:   cd.EveryNWeeks,
				ReferenceDate: cd.ReferenceDate,
			})
		}
		locs = append(locs, locationInfo{
			Label:          loc.Label,
			Scraper:        loc.Scraper,
			PostCode:       loc.PostCode,
			CollectionDays: days,
		})
	}

	resp := listLocationsResponse{Locations: locs}
	return jsonResult(resp)
}

// --- Helpers ---

func resolveDateRange(rangeParam, dateParam string, today time.Time) (time.Time, time.Time, error) {
	if rangeParam != "" {
		switch rangeParam {
		case "today":
			return today, today, nil
		case "tomorrow":
			t := today.AddDate(0, 0, 1)
			return t, t, nil
		case "this_week":
			end := today
			for end.Weekday() != time.Sunday {
				end = end.AddDate(0, 0, 1)
			}
			return today, end, nil
		case "next_week":
			start := today.AddDate(0, 0, 1)
			for start.Weekday() != time.Monday {
				start = start.AddDate(0, 0, 1)
			}
			end := start.AddDate(0, 0, 6) // Sunday
			return start, end, nil
		default:
			return time.Time{}, time.Time{}, fmt.Errorf("invalid range: %q", rangeParam)
		}
	}

	if dateParam != "" {
		d, err := time.Parse("2006-01-02", dateParam)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %q", dateParam)
		}
		return d, d, nil
	}

	// Default: tomorrow
	t := today.AddDate(0, 0, 1)
	return t, t, nil
}

func filterLocations(locations []config.Location, filter string) []config.Location {
	if filter == "" {
		return locations
	}
	var result []config.Location
	for _, loc := range locations {
		if strings.Contains(strings.ToLower(loc.Label), strings.ToLower(filter)) {
			result = append(result, loc)
		}
	}
	return result
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}
