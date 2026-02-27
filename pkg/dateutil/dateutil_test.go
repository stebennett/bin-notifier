package dateutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsTime(t *testing.T) {
	tests := []struct {
		name     string
		day      int
		month    int
		year     int
		expected time.Time
	}{
		{
			name:     "standard date",
			day:      15,
			month:    6,
			year:     2024,
			expected: time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "first day of year",
			day:      1,
			month:    1,
			year:     2024,
			expected: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "last day of year",
			day:      31,
			month:    12,
			year:     2024,
			expected: time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "leap year date",
			day:      29,
			month:    2,
			year:     2024,
			expected: time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := AsTime(test.day, test.month, test.year)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestAsTimeWithMonth(t *testing.T) {
	tests := []struct {
		name     string
		day      int
		month    string
		year     int
		expected time.Time
	}{
		{
			name:     "January",
			day:      15,
			month:    "January",
			year:     2024,
			expected: time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February",
			day:      28,
			month:    "February",
			year:     2024,
			expected: time.Date(2024, time.February, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "March",
			day:      1,
			month:    "March",
			year:     2024,
			expected: time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "December",
			day:      25,
			month:    "December",
			year:     2024,
			expected: time.Date(2024, time.December, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "invalid month defaults to January",
			day:      10,
			month:    "InvalidMonth",
			year:     2024,
			expected: time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := AsTimeWithMonth(test.day, test.month, test.year)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestIsDateMatching(t *testing.T) {
	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "same date matches",
			t1:       time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "same date different times matches",
			t1:       time.Date(2024, time.June, 15, 10, 30, 0, 0, time.UTC),
			t2:       time.Date(2024, time.June, 15, 22, 45, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "different day does not match",
			t1:       time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, time.June, 16, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "different month does not match",
			t1:       time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, time.July, 15, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "different year does not match",
			t1:       time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "year boundary - Dec 31 vs Jan 1",
			t1:       time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsDateMatching(test.t1, test.t2)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestIsOnWeek(t *testing.T) {
	tests := []struct {
		name        string
		reference   time.Time
		target      time.Time
		everyNWeeks int
		expected    bool
	}{
		{
			name:        "same week is on",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday (same day)
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "1 week later is off for fortnightly",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 9, 0, 0, 0, 0, time.UTC),  // Friday +1 week
			everyNWeeks: 2,
			expected:    false,
		},
		{
			name:        "2 weeks later is on for fortnightly",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC), // Friday +2 weeks
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "3 weeks later is off for fortnightly",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 23, 0, 0, 0, 0, time.UTC), // Friday +3 weeks
			everyNWeeks: 2,
			expected:    false,
		},
		{
			name:        "every 3 weeks - week 3 is on",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 23, 0, 0, 0, 0, time.UTC), // Friday +3 weeks
			everyNWeeks: 3,
			expected:    true,
		},
		{
			name:        "every 3 weeks - week 2 is off",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday
			target:      time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC), // Friday +2 weeks
			everyNWeeks: 3,
			expected:    false,
		},
		{
			name:        "weekly is always on",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), // Friday
			target:      time.Date(2026, time.January, 9, 0, 0, 0, 0, time.UTC), // Friday +1 week
			everyNWeeks: 1,
			expected:    true,
		},
		{
			name:        "reference date in future still works",
			reference:   time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC), // Friday +2 weeks
			target:      time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),  // Friday (earlier)
			everyNWeeks: 2,
			expected:    true,
		},
		{
			name:        "reference date in future - off week",
			reference:   time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC), // Friday +2 weeks
			target:      time.Date(2026, time.January, 9, 0, 0, 0, 0, time.UTC),  // Friday +1 week
			everyNWeeks: 2,
			expected:    false,
		},
		{
			name:        "zero everyNWeeks returns false",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			target:      time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			everyNWeeks: 0,
			expected:    false,
		},
		{
			name:        "negative everyNWeeks returns false",
			reference:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			target:      time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			everyNWeeks: -1,
			expected:    false,
		},
		{
			name:        "non-midnight times are normalised",
			reference:   time.Date(2026, time.January, 2, 18, 30, 0, 0, time.UTC),  // Friday 6:30 PM
			target:      time.Date(2026, time.January, 16, 9, 15, 0, 0, time.UTC),  // Friday +2 weeks 9:15 AM
			everyNWeeks: 2,
			expected:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsOnWeek(test.reference, test.target, test.everyNWeeks)
			assert.Equal(t, test.expected, actual)
		})
	}
}

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
