package config

import (
	"errors"

	flags "github.com/jessevdk/go-flags"
)

var (
	ErrHelp = errors.New("help")
)

type Config struct {
	PostCode             string `short:"p" long:"postcode" description:"The postcode to scrape bin times for" required:"true"`
	AddressCode          string `short:"a" long:"addressCode" description:"The address to scrape bin times for" required:"true"`
	RegularCollectionDay int    `short:"r" long:"regularcollectionday" description:"The regular collection day of the week" required:"true"`
	FromNumber           string `short:"f" long:"fromnumber" required:"true" description:"The number to send the confirmation SMS from"`
	ToNumber             string `short:"n" long:"tonumber" required:"true" description:"The number to send the confirmation SMS to"`
	DryRun               bool   `short:"x" long:"dryrun" description:"Run everything, but don't do the booking and assume it succeeds"`
	TodayDate            string `short:"d" long:"todaydate" description:"The date to use for today's date"`
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
