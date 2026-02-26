package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"gopkg.in/yaml.v3"
)

type Flags struct {
	ConfigFile string
	DryRun     bool
	TodayDate  string
}

func ParseFlags(args []string) (Flags, error) {
	fs := flag.NewFlagSet("bin-notifier", flag.ContinueOnError)

	configDefault := os.Getenv("BN_CONFIG_FILE")
	dryRunDefault := os.Getenv("BN_DRY_RUN") == "true"
	todayDateDefault := os.Getenv("BN_TODAY_DATE")

	var f Flags
	fs.StringVar(&f.ConfigFile, "c", configDefault, "path to YAML config file")
	fs.StringVar(&f.ConfigFile, "config", configDefault, "path to YAML config file")
	fs.BoolVar(&f.DryRun, "x", dryRunDefault, "dry-run mode (no SMS sent)")
	fs.BoolVar(&f.DryRun, "dryrun", dryRunDefault, "dry-run mode (no SMS sent)")
	fs.StringVar(&f.TodayDate, "d", todayDateDefault, "override today's date (YYYY-MM-DD)")
	fs.StringVar(&f.TodayDate, "todaydate", todayDateDefault, "override today's date (YYYY-MM-DD)")

	if err := fs.Parse(args); err != nil {
		return Flags{}, err
	}

	if f.ConfigFile == "" {
		return Flags{}, fmt.Errorf("config file is required (-c or BN_CONFIG_FILE)")
	}

	return f, nil
}

type Location struct {
	Label         string       `yaml:"label"`
	Scraper       string       `yaml:"scraper"`
	PostCode      string       `yaml:"postcode"`
	AddressCode   string       `yaml:"address_code"`
	CollectionDay time.Weekday `yaml:"-"`
	RawDay        string       `yaml:"collection_day"`
}

type Config struct {
	FromNumber string     `yaml:"from_number"`
	ToNumber   string     `yaml:"to_number"`
	Locations  []Location `yaml:"locations"`
	DryRun     bool       `yaml:"-"`
	TodayDate  string     `yaml:"-"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if err := validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.FromNumber == "" {
		return fmt.Errorf("from_number is required")
	}
	if cfg.ToNumber == "" {
		return fmt.Errorf("to_number is required")
	}
	if len(cfg.Locations) == 0 {
		return fmt.Errorf("at least one location is required")
	}
	for i := range cfg.Locations {
		loc := &cfg.Locations[i]
		if loc.Label == "" {
			return fmt.Errorf("location %d: label is required", i+1)
		}
		if loc.Scraper == "" {
			return fmt.Errorf("location %d: scraper is required", i+1)
		}
		if loc.PostCode == "" {
			return fmt.Errorf("location %d: postcode is required", i+1)
		}
		if loc.AddressCode == "" {
			return fmt.Errorf("location %d: address_code is required", i+1)
		}
		if loc.RawDay == "" {
			return fmt.Errorf("location %d: collection_day is required", i+1)
		}
		day, err := dateutil.ParseWeekday(loc.RawDay)
		if err != nil {
			return fmt.Errorf("location %d: %w", i+1, err)
		}
		loc.CollectionDay = day
	}
	return nil
}
