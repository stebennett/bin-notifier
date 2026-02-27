package scraper

import (
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"github.com/stretchr/testify/assert"
)

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

func TestParseNextCollectionTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected BinTime
	}{
		{
			name: "food",
			input: `Your next food collection is Monday 26 February 2024
						Your second collection is Monday 26 February 2024
						Your third collection is Monday 4 March 2024`,
			expected: BinTime{"food", dateutil.AsTime(26, 2, 2024)},
		},
		{
			name: "recycling",
			input: `Your next recycling collection is Monday 2 February 2024
						Your second collection is Monday 4 March 2024
						Your second collection is Monday 18 March 2024`,
			expected: BinTime{"recycling", dateutil.AsTime(2, 2, 2024)},
		},
		{
			name: "garden",
			input: `Your next garden collection is Monday 19 February 2024
						Your second collection is Monday 4 March 2024
						Your third collection is Monday 18 March 2024`,
			expected: BinTime{"garden", dateutil.AsTime(19, 2, 2024)},
		},
		{
			name: "refuse",
			input: `Your next refuse collection is Monday 26 February 2024
						Your second collection is Monday 18 March 2024
						Your third collection is Monday 4 April 2024`,
			expected: BinTime{"refuse", dateutil.AsTime(26, 2, 2024)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parseBracknellCollectionTime(test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestParseNextCollectionTime_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "malformed input - no date",
			input: "Your next food collection is Monday",
		},
		{
			name:  "malformed input - wrong format",
			input: "Next collection: food on 26/02/2024",
		},
		{
			name:  "malformed input - missing bin type",
			input: "Your next collection is Monday 26 February 2024",
		},
		{
			name:  "malformed input - missing year",
			input: "Your next food collection is Monday 26 February",
		},
		{
			name:  "random text",
			input: "This is not a valid collection time string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseBracknellCollectionTime(test.input)
			assert.Error(t, err)
			assert.EqualError(t, err, "failed to parse next collection time")
		})
	}
}

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

func TestScrapeBinTimes_ValidationErrors(t *testing.T) {
	scraper := &BracknellScraper{}

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
