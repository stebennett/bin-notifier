package main

import (
	"log"
	"strings"
	"time"

	"github.com/stebennett/bin-notifier/pkg/clients"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
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

	tomorrowsCollections := []string{}

	today := time.Now()
	if config.TodayDate != "" {
		today, err = time.Parse("2006-01-02", config.TodayDate)
		if err != nil {
			log.Fatal(err)
		}
	}

	tomorrow := today.AddDate(0, 0, 1)

	for _, binTime := range binTimes {
		log.Printf("Next collection for %s is %s", binTime.Type, binTime.CollectionTime.String())
		if dateutil.IsDateMatching(binTime.CollectionTime, tomorrow) {
			tomorrowsCollections = append(tomorrowsCollections, binTime.Type)
		}
	}

	twilioClient := clients.NewTwilioClient()
	if len(tomorrowsCollections) != 0 {
		log.Println("Tomorrows collections are:", strings.Join(tomorrowsCollections, ", "))

		_, err = twilioClient.SendSms(config.FromNumber, config.ToNumber, "Tomorrows bin collections are: "+strings.Join(tomorrowsCollections, ", "), config.DryRun)
		if err != nil {
			log.Fatal(err)
		}
	} else if tomorrow.Weekday() == time.Weekday(config.RegularCollectionDay) {
		log.Println("No collections tomorrow, but it's a regular collection day")
		_, err = twilioClient.SendSms(config.FromNumber, config.ToNumber, "Tomorrow is a regular bin collection day, but there are no collections.", config.DryRun)
		if err != nil {

			log.Fatal(err)
		}
	} else {
		log.Println("No collections tomorrow as it's not a regular collection day")
	}

}
