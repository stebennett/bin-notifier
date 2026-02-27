package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/scraper"
	"github.com/stretchr/testify/assert"
)

// mockScraper is a mock implementation of BinScraper for testing
type mockScraper struct {
	binTimes []scraper.BinTime
	err      error
}

func (m *mockScraper) ScrapeBinTimes(postcode string, address string) ([]scraper.BinTime, error) {
	return m.binTimes, m.err
}

// mockSMSClient is a mock implementation of SMSClient for testing
type mockSMSClient struct {
	calls []smsCall
	err   error
}

type smsCall struct {
	from   string
	to     string
	body   string
	dryRun bool
}

func (m *mockSMSClient) SendSms(from string, to string, body string, dryRun bool) error {
	m.calls = append(m.calls, smsCall{from: from, to: to, body: body, dryRun: dryRun})
	return m.err
}

func newMockFactory(scrapers map[string]*mockScraper) ScraperFactory {
	return func(name string) (BinScraper, error) {
		s, ok := scrapers[name]
		if !ok {
			return nil, errors.New("unknown scraper: " + name)
		}
		return s, nil
	}
}

func createTestConfig() config.Config {
	return config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:         time.Tuesday,
						Types:       []string{"General Waste", "Recycling"},
						EveryNWeeks: 1,
					},
				},
			},
		},
	}
}

func TestNotifier_SendsSmsWhenCollectionTomorrow(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	tomorrow := today.AddDate(0, 0, 1)                       // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
			{Type: "Recycling", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	r := results[0]
	assert.Nil(t, r.Error)
	assert.True(t, r.SMSSent)
	assert.Equal(t, 2, len(r.Collections))
	assert.Len(t, mockSMS.calls, 1)
	assert.Equal(t, cfg.FromNumber, mockSMS.calls[0].from)
	assert.Equal(t, cfg.ToNumber, mockSMS.calls[0].to)
	assert.Contains(t, mockSMS.calls[0].body, "General Waste")
	assert.Contains(t, mockSMS.calls[0].body, "Recycling")
}

func TestNotifier_MessagePrefixedWithLabel(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "Home:")
}

func TestNotifier_SendsSmsOnRegularDayNoCollections(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	r := results[0]
	assert.Nil(t, r.Error)
	assert.True(t, r.SMSSent)
	assert.Contains(t, r.Message, "Expected General Waste, Recycling collection tomorrow (Tuesday) but none scheduled.")
	assert.Equal(t, 0, len(r.Collections))
}

func TestNotifier_NoSmsWhenNoCollectionsAndNotRegularDay(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.Locations[0].CollectionDays = []config.CollectionDay{
		{Day: time.Wednesday, Types: []string{"General Waste"}, EveryNWeeks: 1},
	}
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	r := results[0]
	assert.Nil(t, r.Error)
	assert.False(t, r.SMSSent)
	assert.Equal(t, 0, len(r.Collections))
	assert.Len(t, mockSMS.calls, 0)
}

func TestNotifier_ScraperErrorContinuesOtherLocations(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	failScr := &mockScraper{err: errors.New("scraper failed")}
	okScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "Recycling", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{
			"bracknell": failScr,
			"wokingham": okScr,
		}),
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "RG12 1AB", AddressCode: "12345", CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
			{Label: "Office", Scraper: "wokingham", PostCode: "RG42 2XY", AddressCode: "67890", CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 2)
	assert.NotNil(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "scraper failed")
	assert.Nil(t, results[1].Error)
	assert.True(t, results[1].SMSSent)
	assert.Len(t, mockSMS.calls, 1)
}

func TestNotifier_SmsErrorRecordedInResult(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{err: errors.New("SMS send failed")}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "SMS send failed")
}

func TestNotifier_UsesTodayDateFromConfig(t *testing.T) {
	clock := func() time.Time { return time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) }
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          clock,
	}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15"
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.True(t, results[0].SMSSent)
	assert.Contains(t, results[0].Message, "General Waste")
}

func TestNotifier_InvalidTodayDateReturnsError(t *testing.T) {
	mockScr := &mockScraper{binTimes: []scraper.BinTime{}}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return time.Now() },
	}

	cfg := createTestConfig()
	cfg.TodayDate = "invalid-date"
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "invalid today date")
}

func TestNotifier_DryRunPassedToSmsClient(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.DryRun = true
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.Len(t, mockSMS.calls, 1)
	assert.True(t, mockSMS.calls[0].dryRun)
}

func TestNotifier_MultipleLocations(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	homeScraper := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	officeScraper := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "Recycling", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{
			"bracknell": homeScraper,
			"wokingham": officeScraper,
		}),
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{Label: "Home", Scraper: "bracknell", PostCode: "RG12 1AB", AddressCode: "12345", CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
			{Label: "Office", Scraper: "wokingham", PostCode: "RG42 2XY", AddressCode: "67890", CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 2)
	assert.Nil(t, results[0].Error)
	assert.Nil(t, results[1].Error)
	assert.True(t, results[0].SMSSent)
	assert.True(t, results[1].SMSSent)
	assert.Len(t, mockSMS.calls, 2)
	assert.Contains(t, mockSMS.calls[0].body, "Home:")
	assert.Contains(t, mockSMS.calls[1].body, "Office:")
}

func TestNotifier_UnknownScraperRecordsError(t *testing.T) {
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return time.Now() },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{Label: "Home", Scraper: "unknown", PostCode: "RG12 1AB", AddressCode: "12345", CollectionDays: []config.CollectionDay{{Day: time.Tuesday, Types: []string{"General Waste"}, EveryNWeeks: 1}}},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "scraper error")
}

func TestNotifier_FortnightlyOnWeekSendsWarning(t *testing.T) {
	refDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) // Friday
	today := refDate.AddDate(0, 0, 13)                       // Thursday, 2 weeks - 1 day later
	// tomorrow is Friday, 2 weeks after reference = on week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:           time.Friday,
						Types:         []string{"Garden Waste"},
						EveryNWeeks:   2,
						ReferenceDate: "2026-01-02",
					},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.True(t, results[0].SMSSent)
	assert.Contains(t, results[0].Message, "Expected Garden Waste collection tomorrow (Friday)")
}

func TestNotifier_FortnightlyOffWeekNoMessage(t *testing.T) {
	refDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) // Friday
	today := refDate.AddDate(0, 0, 6)                        // Thursday, 1 week - 1 day later
	// tomorrow is Friday, 1 week after reference = off week for fortnightly

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{
						Day:           time.Friday,
						Types:         []string{"Garden Waste"},
						EveryNWeeks:   2,
						ReferenceDate: "2026-01-02",
					},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.False(t, results[0].SMSSent)
	assert.Len(t, mockSMS.calls, 0)
}

func TestNotifier_MultipleCollectionDaysWarnings(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		Clock:          func() time.Time { return today },
	}

	cfg := config.Config{
		FromNumber: "+1234567890",
		ToNumber:   "+0987654321",
		Locations: []config.Location{
			{
				Label:       "Home",
				Scraper:     "bracknell",
				PostCode:    "RG12 1AB",
				AddressCode: "12345",
				CollectionDays: []config.CollectionDay{
					{Day: time.Tuesday, Types: []string{"Recycling"}, EveryNWeeks: 1},
					{Day: time.Tuesday, Types: []string{"Food Waste"}, EveryNWeeks: 1},
				},
			},
		},
	}

	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Nil(t, results[0].Error)
	assert.True(t, results[0].SMSSent)
	assert.Len(t, mockSMS.calls, 2)
	assert.Contains(t, mockSMS.calls[0].body, "Recycling")
	assert.Contains(t, mockSMS.calls[1].body, "Food Waste")
}
