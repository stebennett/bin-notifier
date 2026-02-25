package config

import (
	"fmt"
	"os"
	"time"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"gopkg.in/yaml.v3"
)

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
