package scraper

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
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
	if len(postCode) == 0 {
		return []BinTime{}, errors.New("no postcode specified")
	}
	if len(addressCode) == 0 {
		return []BinTime{}, errors.New("no address specified")
	}

	log.Printf("creating temp user data dir")
	dir, err := os.MkdirTemp("", "chromedp-wokingham")
	if err != nil {
		return []BinTime{}, err
	}
	defer os.RemoveAll(dir)

	log.Printf("setting chrome defaults")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", true),
		chromedp.NoSandbox,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	taskCtx, cancel = context.WithTimeout(taskCtx, 60*time.Second)
	defer cancel()

	log.Printf("navigating to wokingham waste collection page")

	// Step 1: Navigate and accept cookies
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("https://www.wokingham.gov.uk/rubbish-and-recycling/waste-collection/find-your-bin-collection-day"),
		chromedp.WaitVisible(`#edit-postcode-search-csv`, chromedp.ByQuery),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Accept cookies (ignore error if no banner)
	chromedp.Run(taskCtx,
		chromedp.Click(`.agree-button`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)

	// Step 2: Enter postcode and submit
	log.Printf("entering postcode: %s", postCode)
	err = chromedp.Run(taskCtx,
		chromedp.SetValue(`#edit-postcode-search-csv`, postCode, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`#edit-find-address`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Step 3: Select address and show collection dates
	log.Printf("selecting address: %s", addressCode)
	err = chromedp.Run(taskCtx,
		chromedp.WaitVisible(`#edit-address-options-csv`, chromedp.ByQuery),
		chromedp.SetValue(`#edit-address-options-csv`, addressCode, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`#edit-show-collection-dates-csv`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		return []BinTime{}, err
	}

	// Step 4: Wait for results and extract card data
	log.Printf("extracting collection dates")
	var cardCount int
	err = chromedp.Run(taskCtx,
		chromedp.WaitVisible(`.cards-list`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll('.card--waste').length`, &cardCount),
	)
	if err != nil {
		return []BinTime{}, err
	}

	if cardCount == 0 {
		return []BinTime{}, errors.New("no collection cards found")
	}

	// Extract headings and dates from each card
	var headings, dates []string
	err = chromedp.Run(taskCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.card--waste h3')).map(h => h.textContent.trim())`, &headings),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.card--waste .card__date')).map(d => d.textContent.trim())`, &dates),
	)
	if err != nil {
		return []BinTime{}, err
	}

	if len(headings) != len(dates) {
		return []BinTime{}, errors.New("mismatched headings and dates count")
	}

	log.Printf("found %d collection cards, parsing", len(headings))

	binTimes := make([]BinTime, 0, len(headings))
	for i := range headings {
		bt, err := parseWokinghamCollection(headings[i], dates[i])
		if err != nil {
			return binTimes, err
		}
		binTimes = append(binTimes, bt)
	}

	return binTimes, nil
}
