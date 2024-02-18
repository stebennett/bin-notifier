package main

import (
	"log"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/scraper"
)

func main() {
	config, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Scraping bin times for %s - %s", config.AddressCode, config.PostCode)

	scr := scraper.NewBinTimesScraper()
	binTimes, err := scr.ScrapeBinTimes(config.PostCode, config.AddressCode)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(binTimes)
}
