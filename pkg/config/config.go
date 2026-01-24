package config

import (
	"errors"

	flags "github.com/jessevdk/go-flags"
)

var (
	ErrHelp = errors.New("help")
)

type Config struct {
	PostCode             string `short:"p" long:"postcode" env:"BN_POSTCODE" description:"The postcode to scrape bin times for" required:"true"`
	AddressCode          string `short:"a" long:"addressCode" env:"BN_ADDRESS_CODE" description:"The address to scrape bin times for" required:"true"`
	RegularCollectionDay int    `short:"r" long:"regularcollectionday" env:"BN_REGULAR_COLLECTION_DAY" description:"The regular collection day of the week" required:"true"`
	AppriseURL           string `short:"u" long:"appriseurl" env:"BN_APPRISE_URL" required:"true" description:"The Apprise URL for sending notifications"`
	AppriseTag           string `short:"t" long:"apprisetag" env:"BN_APPRISE_TAG" description:"Optional Apprise tag to filter notification services"`
	DryRun               bool   `short:"x" long:"dryrun" env:"BN_DRY_RUN" description:"Run everything, but don't do the booking and assume it succeeds"`
	TodayDate            string `short:"d" long:"todaydate" env:"BN_TODAY_DATE" description:"The date to use for today's date"`
}

func GetConfig() (Config, error) {
	var c Config
	parser := flags.NewParser(&c, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if isErrHelp(err) {
			return c, ErrHelp
		}
		return c, err
	}

	return c, nil
}

func isErrHelp(err error) bool {
	var flagsErr *flags.Error
	if errors.As(err, &flagsErr) {
		return flagsErr.Type == flags.ErrHelp
	}
	return false
}
