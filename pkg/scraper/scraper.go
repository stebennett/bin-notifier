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
	"github.com/chromedp/chromedp/kb"
	"github.com/stebennett/bin-notifier/pkg/dateutil"
	regexputil "github.com/stebennett/bin-notifier/pkg/regexp"
)

type BinTime struct {
	Type           string
	CollectionTime time.Time
}

type BinTimesScraper struct{}

func NewBinTimesScraper() *BinTimesScraper {
	return &BinTimesScraper{}
}

func (s BinTimesScraper) ScrapeBinTimes(postCode string, addressCode string) ([]BinTime, error) {
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
		chromedp.Flag("headless", false),
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
	collectionTimes := make([]string, 4)

	err = chromedp.Run(taskCtx,
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
		chromedp.EvaluateAsDevTools(`document.querySelector("select").dispatchEvent(new Event("change"))`, nil),

		chromedp.WaitVisible(`//h2[@class="collectionHeading"]`),

		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[1]/tr/td[2]`, &collectionTimes[0], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[2]/tr/td[2]`, &collectionTimes[1], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[3]/tr/td[2]`, &collectionTimes[2], chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[4]/tr/td[2]`, &collectionTimes[3], chromedp.NodeVisible, chromedp.BySearch),
	)

	if err != nil {
		return []BinTime{}, err
	}

	log.Printf("bin times collected. parsing to extract.")

	binTimes := make([]BinTime, len(collectionTimes))
	for i, t := range collectionTimes {
		binTimes[i], err = parseNextCollectionTime(t)
		if err != nil {
			return binTimes, err
		}
	}

	return binTimes, nil
}

func parseNextCollectionTime(times string) (BinTime, error) {
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
