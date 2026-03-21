package schedule

import (
	"sort"
	"time"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
)

// Collection represents a projected bin collection for a specific date and location.
type Collection struct {
	Date     time.Time `json:"date"`
	Location string    `json:"location"`
	Types    []string  `json:"types"`
}

// ProjectCollections returns all projected collections for the given locations
// within the date range [from, to] inclusive, sorted by date.
func ProjectCollections(locations []config.Location, from, to time.Time) []Collection {
	var results []Collection

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		for _, loc := range locations {
			var types []string
			for _, cd := range loc.CollectionDays {
				if d.Weekday() != cd.Day {
					continue
				}
				if cd.EveryNWeeks > 1 {
					refDate, err := time.Parse("2006-01-02", cd.ReferenceDate)
					if err != nil {
						continue
					}
					if !dateutil.IsOnWeek(refDate, d, cd.EveryNWeeks) {
						continue
					}
				}
				types = append(types, cd.Types...)
			}
			if len(types) > 0 {
				results = append(results, Collection{
					Date:     d,
					Location: loc.Label,
					Types:    types,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Date.Equal(results[j].Date) {
			return results[i].Location < results[j].Location
		}
		return results[i].Date.Before(results[j].Date)
	})

	return results
}
