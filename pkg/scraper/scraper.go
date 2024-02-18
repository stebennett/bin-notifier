package scraper

import (
	"context"
	"errors"
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

func (s BinTimesScraper) ScrapeBinTimes(postCode string, addressCode string) (BinTimes, error) {
	if len(postCode) == 0 {
		return BinTimes{}, errors.New("no postcode specified")
	}
	if len(addressCode) == 0 {
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
	var foodTimes string
	var blueTimes string
	var greenTimes string
	var brownTimes string

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

		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[1]/tr/td[2]`, &foodTimes, chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[2]/tr/td[2]`, &blueTimes, chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[3]/tr/td[2]`, &brownTimes, chromedp.NodeVisible, chromedp.BySearch),
		chromedp.Text(`//table[@class="bin-table"]/tr[2]/table/table[4]/tr/td[2]`, &greenTimes, chromedp.NodeVisible, chromedp.BySearch),
	)

	log.Printf("food times: %s", foodTimes)
	log.Printf("blue times: %s", blueTimes)
	log.Printf("brown times: %s", brownTimes)
	log.Printf("green times: %s", greenTimes)

	if err != nil {
		return BinTimes{}, err
	}

	return BinTimes{}, errors.New("failed to get bin times")
}
