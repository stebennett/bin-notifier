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
    collection_days:
      - day: tuesday
        types: ["Recycling", "General Waste"]
  - label: Office
    scraper: wokingham
    postcode: "RG42 2XY"
    address_code: "67890"
    collection_days:
      - day: thursday
        types: ["Recycling"]
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
	assert.Equal(t, time.Tuesday, cfg.Locations[0].CollectionDays[0].Day)
	assert.Equal(t, []string{"Recycling", "General Waste"}, cfg.Locations[0].CollectionDays[0].Types)
	assert.Equal(t, 1, cfg.Locations[0].CollectionDays[0].EveryNWeeks)

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
    collection_days:
      - day: WEDNESDAY
        types: ["General Waste"]
`)

	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, time.Wednesday, cfg.Locations[0].CollectionDays[0].Day)
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
    collection_days:
      - day: notaday
        types: ["General Waste"]
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
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
    collection_days:
      - day: tuesday
        types: ["General Waste"]`,
			errText: "address_code is required",
		},
		{
			name: "missing collection_days",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"`,
			errText: "collection_days must have at least one entry",
		},
		{
			name: "empty collection_days",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days: []`,
			errText: "collection_days must have at least one entry",
		},
		{
			name: "missing types",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday`,
			errText: "types must have at least one entry",
		},
		{
			name: "empty types",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: []`,
			errText: "types must have at least one entry",
		},
		{
			name: "missing reference_date for fortnightly",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling"]
        every_n_weeks: 2`,
			errText: "reference_date is required",
		},
		{
			name: "invalid reference_date format",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling"]
        every_n_weeks: 2
        reference_date: "not-a-date"`,
			errText: "invalid reference_date",
		},
		{
			name: "reference_date weekday mismatch",
			yaml: `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling"]
        every_n_weeks: 2
        reference_date: "2026-01-07"`,
			errText: "reference_date must fall on",
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

func TestLoadConfig_FortnightlySchedule(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling", "General Waste"]
      - day: friday
        every_n_weeks: 2
        reference_date: "2026-01-02"
        types: ["Garden Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Len(t, cfg.Locations[0].CollectionDays, 2)
	cd0 := cfg.Locations[0].CollectionDays[0]
	assert.Equal(t, time.Tuesday, cd0.Day)
	assert.Equal(t, []string{"Recycling", "General Waste"}, cd0.Types)
	assert.Equal(t, 1, cd0.EveryNWeeks)
	assert.Equal(t, "", cd0.ReferenceDate)
	cd1 := cfg.Locations[0].CollectionDays[1]
	assert.Equal(t, time.Friday, cd1.Day)
	assert.Equal(t, []string{"Garden Waste"}, cd1.Types)
	assert.Equal(t, 2, cd1.EveryNWeeks)
	assert.Equal(t, "2026-01-02", cd1.ReferenceDate)
}

func TestLoadConfig_EveryNWeeksDefaultsTo1(t *testing.T) {
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["Recycling"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, 1, cfg.Locations[0].CollectionDays[0].EveryNWeeks)
}

func TestParseFlags_ConfigFromFlag(t *testing.T) {
	flags, err := ParseFlags([]string{"-c", "/path/to/config.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "/path/to/config.yaml", flags.ConfigFile)
	assert.False(t, flags.DryRun)
	assert.Equal(t, "", flags.TodayDate)
}

func TestParseFlags_AllFlags(t *testing.T) {
	flags, err := ParseFlags([]string{"-c", "/path/to/config.yaml", "-x", "-d", "2024-01-15"})
	assert.NoError(t, err)
	assert.Equal(t, "/path/to/config.yaml", flags.ConfigFile)
	assert.True(t, flags.DryRun)
	assert.Equal(t, "2024-01-15", flags.TodayDate)
}

func TestParseFlags_LongFlags(t *testing.T) {
	flags, err := ParseFlags([]string{"--config", "/path/to/config.yaml", "--dryrun", "--todaydate", "2024-01-15"})
	assert.NoError(t, err)
	assert.Equal(t, "/path/to/config.yaml", flags.ConfigFile)
	assert.True(t, flags.DryRun)
	assert.Equal(t, "2024-01-15", flags.TodayDate)
}

func TestParseFlags_MissingConfigReturnsError(t *testing.T) {
	_, err := ParseFlags([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file is required")
}

func TestParseFlags_ConfigFromEnv(t *testing.T) {
	t.Setenv("BN_CONFIG_FILE", "/env/config.yaml")
	flags, err := ParseFlags([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "/env/config.yaml", flags.ConfigFile)
}

func TestParseFlags_DryRunFromEnv(t *testing.T) {
	t.Setenv("BN_CONFIG_FILE", "/env/config.yaml")
	t.Setenv("BN_DRY_RUN", "true")
	flags, err := ParseFlags([]string{})
	assert.NoError(t, err)
	assert.True(t, flags.DryRun)
}

func TestParseFlags_TodayDateFromEnv(t *testing.T) {
	t.Setenv("BN_CONFIG_FILE", "/env/config.yaml")
	t.Setenv("BN_TODAY_DATE", "2024-01-15")
	flags, err := ParseFlags([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "2024-01-15", flags.TodayDate)
}

func TestParseFlags_FlagOverridesEnv(t *testing.T) {
	t.Setenv("BN_CONFIG_FILE", "/env/config.yaml")
	flags, err := ParseFlags([]string{"-c", "/flag/config.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "/flag/config.yaml", flags.ConfigFile)
}

func TestLoadConfig_FromNumberFromEnv(t *testing.T) {
	t.Setenv("BN_FROM_NUMBER", "+440000000000")
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+440000000000", cfg.FromNumber)
	assert.Equal(t, "+449876543210", cfg.ToNumber)
}

func TestLoadConfig_ToNumberFromEnv(t *testing.T) {
	t.Setenv("BN_TO_NUMBER", "+440000000000")
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+441234567890", cfg.FromNumber)
	assert.Equal(t, "+440000000000", cfg.ToNumber)
}

func TestLoadConfig_PhoneNumbersFromEnvOverrideEmptyYAML(t *testing.T) {
	t.Setenv("BN_FROM_NUMBER", "+440000000000")
	t.Setenv("BN_TO_NUMBER", "+441111111111")
	path := writeConfigFile(t, `
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+440000000000", cfg.FromNumber)
	assert.Equal(t, "+441111111111", cfg.ToNumber)
}
