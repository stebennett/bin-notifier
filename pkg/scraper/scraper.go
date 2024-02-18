package scraper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type BinTimes struct {
	Green time.Time
	Blue  time.Time
	Brown time.Time
	Food  time.Time
}

type BinTimesScraper struct{}

func NewBinTimesScraper() *BinTimesScraper {
	return &BinTimesScraper{}
}

func (s BinTimesScraper) ScrapeBinTimes(postCode string, address string) (BinTimes, error) {
	if len(postCode) == 0 {
		return BinTimes{}, errors.New("no postcode specified")
	}
	if len(address) == 0 {
		return BinTimes{}, errors.New("no address specified")
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
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithDebugf(log.Printf))
	defer cancel()

	log.Printf("creating timeout")
	taskCtx, cancel = context.WithTimeout(taskCtx, 120*time.Second)
	defer cancel()

	log.Printf("running task")
	//var binTimes BinTimes
	err = chromedp.Run(taskCtx,
		chromedp.Navigate("https://selfservice.mybfc.bracknell-forest.gov.uk/w/webpage/waste-collection-days"),

		chromedp.WaitVisible(`//a[text()="Accept all cookies"]`),
		chromedp.Click(`//a[text()="Accept all cookies"]`),

		chromedp.WaitVisible(`input[placeholder="Postcode or street name"]`),
		chromedp.SendKeys(`input[placeholder="Postcode or street name"]`, postCode),
		chromedp.SendKeys(`input[placeholder="Postcode or street name"]`, kb.Enter),
		chromedp.WaitVisible(`//select`),
		chromedp.Click(fmt.Sprintf(`//select/option[text()="%s"]`, address)),

		chromedp.WaitVisible(`div[class="collectionHeading"]`),
	)

	if err != nil {
		return BinTimes{}, err
	}

	return BinTimes{}, errors.New("failed to get bin times")
}
