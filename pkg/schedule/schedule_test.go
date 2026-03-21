package schedule

import (
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestProjectCollections_WeeklySchedule(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling", "General Waste"}, EveryNWeeks: 1},
			},
		},
	}

	from := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC) // Monday
	to := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)   // Sunday

	results := ProjectCollections(locations, from, to)

	assert.Len(t, results, 1)
	assert.Equal(t, time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC), results[0].Date)
	assert.Equal(t, "Home", results[0].Location)
	assert.Equal(t, []string{"Recycling", "General Waste"}, results[0].Types)
}

func TestProjectCollections_FortnightlySchedule(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{
					Day:           time.Friday,
					Types:         []string{"Garden Waste"},
					EveryNWeeks:   2,
					ReferenceDate: "2026-01-02",
				},
			},
		},
	}

	// 2026-01-02 is the reference (on-week). 2 weeks later = 2026-01-16 (on).
	// 1 week later = 2026-01-09 (off).
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)  // Monday
	to := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)    // Sunday

	results := ProjectCollections(locations, from, to)

	assert.Len(t, results, 1)
	assert.Equal(t, time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), results[0].Date)
	assert.Equal(t, []string{"Garden Waste"}, results[0].Types)
}

func TestProjectCollections_MultipleLocations(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
			},
		},
		{
			Label: "Office",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1},
			},
		},
	}

	from := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC) // Tuesday
	to := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)   // Tuesday

	results := ProjectCollections(locations, from, to)

	assert.Len(t, results, 2)
	assert.Equal(t, "Home", results[0].Location)
	assert.Equal(t, "Office", results[1].Location)
}

func TestProjectCollections_EmptyRange(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
			},
		},
	}

	// from after to — empty range
	from := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	results := ProjectCollections(locations, from, to)
	assert.Empty(t, results)
}

func TestProjectCollections_NoMatchingDays(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Friday, Types: []string{"Recycling"}, EveryNWeeks: 1},
			},
		},
	}

	// Monday to Thursday — no Friday
	from := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	results := ProjectCollections(locations, from, to)
	assert.Empty(t, results)
}

func TestProjectCollections_MultipleCollectionDaysSameDay(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
				{Day: time.Tuesday, Types: []string{"Food Waste"}, EveryNWeeks: 1},
			},
		},
	}

	from := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC) // Tuesday
	to := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)

	results := ProjectCollections(locations, from, to)

	assert.Len(t, results, 1)
	assert.Equal(t, []string{"Recycling", "Food Waste"}, results[0].Types)
}

func TestProjectCollections_SortedByDate(t *testing.T) {
	locations := []config.Location{
		{
			Label: "Home",
			CollectionDays: []config.CollectionDay{
				{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
				{Day: time.Friday, Types: []string{"Garden Waste"}, EveryNWeeks: 1},
			},
		},
	}

	from := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC) // Monday
	to := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)   // Sunday

	results := ProjectCollections(locations, from, to)

	assert.Len(t, results, 2)
	assert.True(t, results[0].Date.Before(results[1].Date))
}
