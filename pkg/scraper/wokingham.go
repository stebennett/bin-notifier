package scraper

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/stebennett/bin-notifier/pkg/dateutil"
)

type WokinghamScraper struct{}

func parseWokinghamCollection(heading string, dateText string) (BinTime, error) {
	heading = strings.TrimSpace(heading)
	dateText = strings.TrimSpace(dateText)

	if len(heading) == 0 {
		return BinTime{}, errors.New("empty heading")
	}

	// Extract bin type, stripping "(week N)" suffix if present
	typeExp := regexp.MustCompile(`^(?P<BinType>[A-Za-z\s]+?)\s*(?:\(week \d+\))?$`)
	typeMatches := typeExp.FindStringSubmatch(heading)
	if typeMatches == nil {
		return BinTime{}, errors.New("failed to parse bin type from heading")
	}
	binType := strings.TrimSpace(typeMatches[1])

	// Extract DD/MM/YYYY from date text
	dateExp := regexp.MustCompile(`(?P<Day>\d{2})/(?P<Month>\d{2})/(?P<Year>\d{4})`)
	dateMatches := dateExp.FindStringSubmatch(dateText)
	if dateMatches == nil {
		return BinTime{}, errors.New("failed to parse date from date text")
	}

	day, _ := strconv.Atoi(dateMatches[1])
	month, _ := strconv.Atoi(dateMatches[2])
	year, _ := strconv.Atoi(dateMatches[3])

	return BinTime{
		Type:           binType,
		CollectionTime: dateutil.AsTime(day, month, year),
	}, nil
}

func (s *WokinghamScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	return nil, errors.New("wokingham scraper not implemented")
}
