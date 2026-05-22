package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/apiclient"
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

// mockAPIClient is a mock implementation of APIClient for testing
type mockAPIClient struct {
	calls []apiCall
	fail  bool
}

type apiCall struct {
	Label     string
	ScrapedAt time.Time
	Items     []apiclient.Collection
}

func (m *mockAPIClient) PushCollections(label string, scrapedAt time.Time, items []apiclient.Collection) error {
	m.calls = append(m.calls, apiCall{Label: label, ScrapedAt: scrapedAt, Items: items})
	if m.fail {
		return fmt.Errorf("simulated api failure")
	}
	return nil
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

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
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
		Clock:          func() time.Time { return today },
	}

	cfg := createTestConfig()
	results := notifier.Run(cfg)

	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "Home:")
}

func TestNotifier_SendsSmsOnRegularDayNoCollections(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	nextWeek := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC) // Monday +1 week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	nextWeek := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC) // Monday +1 week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

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
		APIClient: noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{err: errors.New("SMS send failed")}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
		APIClient:      noopAPIClient{},
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
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

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
		APIClient: noopAPIClient{},
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
		APIClient:      noopAPIClient{},
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
	today := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) // Thursday
	// tomorrow is Friday 2026-01-16, exactly 2 weeks after reference date 2026-01-02 = on week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
	today := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC) // Thursday
	// tomorrow is Friday 2026-01-09, 1 week after reference date 2026-01-02 = off week for fortnightly

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)   // Monday
	nextWeek := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC) // Monday +1 week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      noopAPIClient{},
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

func TestNotifier_PushesScrapedDataBeforeSMS(t *testing.T) {
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday
	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{{Type: "General Waste", CollectionTime: tomorrow}},
	}
	mockSMS := &mockSMSClient{}
	mockAPI := &mockAPIClient{}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15"

	n := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      mockAPI,
		Clock:          func() time.Time { return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) },
	}
	results := n.Run(cfg)

	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error)
	assert.Len(t, mockAPI.calls, 1, "API push should have been attempted")
	assert.Equal(t, "Home", mockAPI.calls[0].Label)
	assert.Equal(t, []apiclient.Collection{{BinType: "General Waste", Date: "2024-01-16"}}, mockAPI.calls[0].Items)
	assert.NotEmpty(t, mockSMS.calls, "SMS should still be sent")
}

func TestNotifierDryRunSkipsPush(t *testing.T) {
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC) // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}
	mockAPI := &mockAPIClient{}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15"
	cfg.DryRun = true

	n := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      mockAPI,
		Clock:          func() time.Time { return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) },
	}
	results := n.Run(cfg)

	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error)
	assert.Empty(t, mockAPI.calls, "dry-run must not push to the API")
	assert.NotEmpty(t, mockSMS.calls, "SMS should still be sent in dry-run mode")
}

func TestNotifier_PushFailureDoesNotBlockSMS(t *testing.T) {
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{{Type: "General Waste", CollectionTime: tomorrow}},
	}
	mockSMS := &mockSMSClient{}
	mockAPI := &mockAPIClient{fail: true}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15"

	n := &Notifier{
		ScraperFactory: newMockFactory(map[string]*mockScraper{"bracknell": mockScr}),
		SMSClient:      mockSMS,
		APIClient:      mockAPI,
		Clock:          func() time.Time { return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) },
	}
	results := n.Run(cfg)

	assert.Len(t, results, 1)
	assert.NoError(t, results[0].Error, "push failure must not surface as a notifier error")
	assert.NotEmpty(t, mockSMS.calls, "SMS should still be sent despite push failure")
}
