package scraper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
	regexputil "github.com/stebennett/bin-notifier/pkg/regexp"
)

type BracknellScraper struct{}

func (s *BracknellScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
	if len(postCode) == 0 {
		return []BinTime{}, errors.New("no postcode specified")
	}
	if len(addressCode) == 0 {
		return []BinTime{}, errors.New("no address specified")
	}

	log.Printf("creating temp user data dir")
	dir, err := os.MkdirTemp("", "chromedp-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	log.Printf("setting chrome defaults")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
		chromedp.Flag("headless", true),
		chromedp.NoSandbox,
	)

	log.Printf("creating chrome context")
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	log.Printf("creating logger")
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	log.Printf("creating timeout")
	taskCtx, cancel = context.WithTimeout(taskCtx, 60*time.Second)
	defer cancel()

	log.Printf("running task")
	err = chromedp.Do(taskCtx,
		chromedp.Navigate("https://selfservice.mybfc.bracknell-forest.gov.uk/w/webpage/waste-collection-days"),

		chromedp.WaitVisible(`//a[text()="Accept all cookies"]`),
		chromedp.Click(`//a[text()="Accept all cookies"]`),
		chromedp.WaitNotVisible(`//a[text()="Accept all cookies"]`),

		chromedp.SendKeys(`input[type="text"]`, postCode),
		chromedp.Sleep(2*time.Second),

		chromedp.SendKeys(`input[type="text"]`, kb.Enter),
		chromedp.Sleep(2*time.Second),

		chromedp.WaitVisible(`//select`),
		chromedp.SetValue(`//select`, addressCode),
		chromedp.Sleep(2*time.Second),
		chromedp.EvaluateAsDevTools[chromedp.Void](`document.querySelector("select").dispatchEvent(new Event("change"))`),

		chromedp.WaitVisible(`//h2[@class="collectionHeading"]`),
	)
	if err != nil {
		return []BinTime{}, err
	}

	collectionTimes := make([]string, 4)
	for i := range collectionTimes {
		sel := fmt.Sprintf(`//table[@class="bin-table"]/tr[2]/table/table[%d]/tr/td[2]`, i+1)
		collectionTimes[i], err = chromedp.Run(taskCtx, chromedp.Text(sel, chromedp.NodeVisible))
		if err != nil {
			return []BinTime{}, err
		}
	}

	log.Printf("bin times collected. parsing to extract.")

	binTimes := make([]BinTime, len(collectionTimes))
	for i, t := range collectionTimes {
		binTimes[i], err = parseBracknellCollectionTime(t)
		if err != nil {
			return binTimes, err
		}
	}

	return binTimes, nil
}

func parseBracknellCollectionTime(times string) (BinTime, error) {
	t := strings.Split(times, "\n")

	exp := `Your next (?P<BinType>[a-z\s]+) collection is [A-Za-z]+ (?P<Date>\d+) (?P<Month>[A-Za-z]+) (?P<Year>\d{4})`
	re := regexp.MustCompile(exp)

	matches := regexputil.FindNamedMatches(re, t[0])
	if len(matches) != 4 {
		return BinTime{}, errors.New("failed to parse next collection time")
	}

	day, _ := strconv.Atoi(matches["Date"])
	year, _ := strconv.Atoi(matches["Year"])

	return BinTime{matches["BinType"], dateutil.AsTimeWithMonth(day, matches["Month"], year)}, nil
}
