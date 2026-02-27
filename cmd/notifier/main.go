package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/stebennett/bin-notifier/pkg/clients"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
	"github.com/stebennett/bin-notifier/pkg/scraper"
)

// ScraperFactory resolves a BinScraper by name.
type ScraperFactory func(name string) (BinScraper, error)

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
	ScraperFactory ScraperFactory
	SMSClient      SMSClient
	Clock          func() time.Time
}

// NotificationResult contains the result of a notification run for a single location.
type NotificationResult struct {
	Label       string
	Collections []string
	SMSSent     bool
	Message     string
	Error       error
}

// Run executes the notification workflow for all locations in the config.
func (n *Notifier) Run(cfg config.Config) []NotificationResult {
	today := n.Clock()
	if cfg.TodayDate != "" {
		parsed, err := time.Parse("2006-01-02", cfg.TodayDate)
		if err != nil {
			return []NotificationResult{{Error: fmt.Errorf("invalid today date: %w", err)}}
		}
		today = parsed
	}

	results := make([]NotificationResult, 0, len(cfg.Locations))
	for _, loc := range cfg.Locations {
		result := n.processLocation(cfg, loc, today)
		results = append(results, result)
	}
	return results
}

func (n *Notifier) processLocation(cfg config.Config, loc config.Location, today time.Time) NotificationResult {
	result := NotificationResult{Label: loc.Label}

	log.Printf("[%s] Scraping bin times for %s - %s", loc.Label, loc.AddressCode, loc.PostCode)

	s, err := n.ScraperFactory(loc.Scraper)
	if err != nil {
		result.Error = fmt.Errorf("[%s] scraper error: %w", loc.Label, err)
		return result
	}

	binTimes, err := s.ScrapeBinTimes(loc.PostCode, loc.AddressCode)
	if err != nil {
		result.Error = fmt.Errorf("[%s] scrape error: %w", loc.Label, err)
		return result
	}

	tomorrow := today.AddDate(0, 0, 1)

	for _, binTime := range binTimes {
		log.Printf("[%s] Next collection for %s is %s", loc.Label, binTime.Type, binTime.CollectionTime.String())
		if dateutil.IsDateMatching(binTime.CollectionTime, tomorrow) {
			result.Collections = append(result.Collections, binTime.Type)
		}
	}

	if len(result.Collections) != 0 {
		result.Message = loc.Label + ": Tomorrows bin collections are: " + strings.Join(result.Collections, ", ")
		log.Printf("[%s] %s", loc.Label, result.Message)

		err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, result.Message, cfg.DryRun)
		if err != nil {
			result.Error = fmt.Errorf("[%s] SMS error: %w", loc.Label, err)
			return result
		}
		result.SMSSent = true
	} else {
		for _, cd := range loc.CollectionDays {
			if tomorrow.Weekday() != cd.Day {
				continue
			}
			if cd.EveryNWeeks > 1 {
				refDate, err := time.Parse("2006-01-02", cd.ReferenceDate)
				if err != nil {
					result.Error = fmt.Errorf("[%s] invalid reference_date in collection schedule: %w", loc.Label, err)
					return result
				}
				if !dateutil.IsOnWeek(refDate, tomorrow, cd.EveryNWeeks) {
					continue
				}
			}
			msg := fmt.Sprintf("%s: Expected %s collection tomorrow (%s) but none scheduled.",
				loc.Label, strings.Join(cd.Types, ", "), tomorrow.Weekday())
			log.Printf("[%s] %s", loc.Label, msg)
			if result.Message != "" {
				result.Message += "; " + msg
			} else {
				result.Message = msg
			}

			err = n.SMSClient.SendSms(cfg.FromNumber, cfg.ToNumber, msg, cfg.DryRun)
			if err != nil {
				result.Error = fmt.Errorf("[%s] SMS error: %w", loc.Label, err)
				return result
			}
			result.SMSSent = true
		}
		if !result.SMSSent {
			log.Printf("[%s] No collections tomorrow and not an expected collection day", loc.Label)
		}
	}

	return result
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
	flags, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	cfg.DryRun = flags.DryRun
	cfg.TodayDate = flags.TodayDate

	notifier := &Notifier{
		ScraperFactory: func(name string) (BinScraper, error) {
			return scraper.NewScraper(name)
		},
		SMSClient: &twilioSMSClientAdapter{client: clients.NewTwilioClient()},
		Clock:     time.Now,
	}

	results := notifier.Run(cfg)
	hasError := false
	for _, r := range results {
		if r.Error != nil {
			log.Printf("ERROR: %v", r.Error)
			hasError = true
		}
	}
	if hasError {
		os.Exit(1)
	}
}
