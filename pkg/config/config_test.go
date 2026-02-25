package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestLoadConfig_ValidMultipleLocations(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday
  - label: Office
    scraper: wokingham
    postcode: "RG42 2XY"
    address_code: "67890"
    collection_day: thursday
`)

	cfg, err := LoadConfig(path)

	assert.NoError(t, err)
	assert.Equal(t, "+441234567890", cfg.FromNumber)
	assert.Equal(t, "+449876543210", cfg.ToNumber)
	assert.Len(t, cfg.Locations, 2)

	assert.Equal(t, "Home", cfg.Locations[0].Label)
	assert.Equal(t, "bracknell", cfg.Locations[0].Scraper)
	assert.Equal(t, "RG12 1AB", cfg.Locations[0].PostCode)
	assert.Equal(t, "12345", cfg.Locations[0].AddressCode)
	assert.Equal(t, time.Tuesday, cfg.Locations[0].CollectionDay)

	assert.Equal(t, "Office", cfg.Locations[1].Label)
	assert.Equal(t, "wokingham", cfg.Locations[1].Scraper)
}

func TestLoadConfig_CaseInsensitiveDay(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: WEDNESDAY
`)

	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, time.Wednesday, cfg.Locations[0].CollectionDay)
}

func TestLoadConfig_InvalidDay(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: notaday
`)

	_, err := LoadConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid weekday")
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		errText string
	}{
		{
			name: "missing from_number",
			yaml: `
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "from_number is required",
		},
		{
			name: "missing to_number",
			yaml: `
from_number: "+441234567890"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "to_number is required",
		},
		{
			name: "no locations",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations: []`,
			errText: "at least one location is required",
		},
		{
			name: "missing label",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "label is required",
		},
		{
			name: "missing scraper",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_day: tuesday`,
			errText: "scraper is required",
		},
		{
			name: "missing postcode",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    address_code: "12345"
    collection_day: tuesday`,
			errText: "postcode is required",
		},
		{
			name: "missing address_code",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    collection_day: tuesday`,
			errText: "address_code is required",
		},
		{
			name: "missing collection_day",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"`,
			errText: "collection_day is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfigFile(t, test.yaml)
			_, err := LoadConfig(path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.errText)
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := writeConfigFile(t, `{{{invalid yaml`)
	_, err := LoadConfig(path)
	assert.Error(t, err)
}
