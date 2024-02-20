package scraper

import (
	"testing"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"github.com/stretchr/testify/assert"
)

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
			actual, err := parseNextCollectionTime(test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
