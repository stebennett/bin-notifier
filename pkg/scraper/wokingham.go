package scraper

import "fmt"

type WokinghamScraper struct{}

func (s *WokinghamScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	return nil, fmt.Errorf("wokingham scraper not implemented")
}
