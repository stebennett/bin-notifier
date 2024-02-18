package config

import (
	"errors"

	flags "github.com/jessevdk/go-flags"
)

var (
	ErrHelp = errors.New("help")
)

type Config struct {
	PostCode    string `short:"p" long:"postcode" description:"The postcode to scrape bin times for" required:"true"`
	AddressCode string `short:"a" long:"addressCode" description:"The address to scrape bin times for" required:"true"`
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
