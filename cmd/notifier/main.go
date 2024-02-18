package main

import (
	"log"

	"github.com/stebennett/bin-notifier/pkg/scraper"
)

func main() {
	scr := scraper.NewBinTimesScraper()

	binTimes, err := scr.ScrapeBinTimes("RG12 8FN", "6 CUCKOO LANE, BRACKNELL, RG12 8FN")

	if err != nil {
		log.Fatal(err)
	}

	log.Println(binTimes)
}
