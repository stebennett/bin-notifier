package regexputil

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindNamedMatches(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		expected map[string]string
	}{
		{
			name:    "single named group",
			pattern: `Hello (?P<name>\w+)`,
			input:   "Hello World",
			expected: map[string]string{
				"name": "World",
			},
		},
		{
			name:    "multiple named groups",
			pattern: `(?P<first>\w+) (?P<second>\w+) (?P<third>\w+)`,
			input:   "one two three",
			expected: map[string]string{
				"first":  "one",
				"second": "two",
				"third":  "three",
			},
		},
		{
			name:    "bin collection pattern",
			pattern: `Your next (?P<BinType>[a-z\s]+) collection is [A-Za-z]+ (?P<Date>\d+) (?P<Month>[A-Za-z]+) (?P<Year>\d{4})`,
			input:   "Your next food collection is Monday 26 February 2024",
			expected: map[string]string{
				"BinType": "food",
				"Date":    "26",
				"Month":   "February",
				"Year":    "2024",
			},
		},
		{
			name:    "mixed named and unnamed groups",
			pattern: `(\d+)-(?P<named>\w+)-(\d+)`,
			input:   "123-test-456",
			expected: map[string]string{
				"named": "test",
			},
		},
		{
			name:     "no named groups returns empty map",
			pattern:  `(\w+) (\w+)`,
			input:    "hello world",
			expected: map[string]string{},
		},
		{
			name:     "no match returns empty map",
			pattern:  `(?P<name>\d+)`,
			input:    "no digits here",
			expected: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			re := regexp.MustCompile(test.pattern)
			actual := FindNamedMatches(re, test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
