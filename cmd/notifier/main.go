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

// BinScraper is an interface for scraping bin collection times.
type BinScraper interface {
	ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error)
}

// SMSClient is an interface for sending SMS messages.
type SMSClient interface {
	SendSms(from string, to string, body string, dryRun bool) error
}

// Notifier orchestrates the bin collection notification workflow.
type Notifier struct {
	Scraper   BinScraper
	SMSClient SMSClient
	Clock     func() time.Time
}

// NotificationResult contains the result of a notification run.
type NotificationResult struct {
	Collections []string
	SMSSent     bool
	Message     string
}

// Run executes the notification workflow with the given configuration.
func (n *Notifier) Run(cfg config.Config) (*NotificationResult, error) {
	log.Printf("Scraping bin times for %s - %s", cfg.AddressCode, cfg.PostCode)

	binTimes, err := n.Scraper.ScrapeBinTimes(cfg.PostCode, cfg.AddressCode)
	if err != nil {
		return nil, err
	}

	today := n.Clock()
	if cfg.TodayDate != "" {
		today, err = time.Parse("2006-01-02", cfg.TodayDate)
		if err != nil {
			return nil, err
		}
	}

	tomorrow := today.AddDate(0, 0, 1)

	tomorrowsCollections := []string{}
	for _, binTime := range binTimes {
		log.Printf("Next collection for %s is %s", binTime.Type, binTime.CollectionTime.String())
		if dateutil.IsDateMatching(binTime.CollectionTime, tomorrow) {
			tomorrowsCollections = append(tomorrowsCollections, binTime.Type)
		}
	}

	result := &NotificationResult{
		Collections: tomorrowsCollections,
	}

	if len(tomorrowsCollections) != 0 {
		result.Message = "Tomorrows bin collections are: " + strings.Join(tomorrowsCollections, ", ")
		log.Println("Tomorrows collections are:", strings.Join(tomorrowsCollections, ", "))

		err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, result.Message, cfg.DryRun)
		if err != nil {
			return nil, err
		}
		result.SMSSent = true
	} else if tomorrow.Weekday() == time.Weekday(cfg.RegularCollectionDay) {
		result.Message = "Tomorrow is a regular bin collection day, but there are no collections."
		log.Println("No collections tomorrow, but it's a regular collection day")

		err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, result.Message, cfg.DryRun)
		if err != nil {
			return nil, err
		}
		result.SMSSent = true
	} else {
		log.Println("No collections tomorrow as it's not a regular collection day")
	}

	return result, nil
}

// twilioSMSClientAdapter adapts TwilioClient to the SMSClient interface
type twilioSMSClientAdapter struct {
	client *clients.TwilioClient
}

func (a *twilioSMSClientAdapter) SendSms(from string, to string, body string, dryRun bool) error {
	_, err := a.client.SendSms(from, to, body, dryRun)
	return err
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	notifier := &Notifier{
		Scraper:   scraper.NewBinTimesScraper(),
		SMSClient: &twilioSMSClientAdapter{client: clients.NewTwilioClient()},
		Clock:     time.Now,
	}

	_, err = notifier.Run(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
