package scraper

import (
	"fmt"
	"strings"
	"time"
)

type BinTime struct {
	Type           string
	CollectionTime time.Time
}

type BinScraper interface {
	ScrapeBinTimes(postcode string, addressCode string) ([]BinTime, error)
}

func NewScraper(name string) (BinScraper, error) {
	switch strings.ToLower(name) {
	case "bracknell":
		return &BracknellScraper{}, nil
	case "wokingham":
		return &WokinghamScraper{}, nil
	default:
		return nil, fmt.Errorf("unknown scraper: %q", name)
	}
}
