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
	sendSmsCalled bool
	lastFrom      string
	lastTo        string
	lastBody      string
	lastDryRun    bool
	err           error
}

func (m *mockSMSClient) SendSms(from string, to string, body string, dryRun bool) error {
	m.sendSmsCalled = true
	m.lastFrom = from
	m.lastTo = to
	m.lastBody = body
	m.lastDryRun = dryRun
	return m.err
}

func createTestConfig() config.Config {
	return config.Config{
		PostCode:             "RG12 1AB",
		AddressCode:          "12345",
		RegularCollectionDay: int(time.Tuesday),
		FromNumber:           "+1234567890",
		ToNumber:             "+0987654321",
		DryRun:               false,
	}
}

func TestNotifier_SendsSmsWhenCollectionTomorrow(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	tomorrow := today.AddDate(0, 0, 1)                     // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
			{Type: "Recycling", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockSMS.sendSmsCalled)
	assert.Equal(t, cfg.FromNumber, mockSMS.lastFrom)
	assert.Equal(t, cfg.ToNumber, mockSMS.lastTo)
	assert.Contains(t, mockSMS.lastBody, "General Waste")
	assert.Contains(t, mockSMS.lastBody, "Recycling")
	assert.True(t, result.SMSSent)
	assert.Equal(t, 2, len(result.Collections))
}

func TestNotifier_SendsSmsOnRegularDayNoCollections(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday (regular collection day), but no collections
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)                     // Collections are next week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.RegularCollectionDay = int(time.Tuesday) // Tomorrow is Tuesday
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockSMS.sendSmsCalled)
	assert.Contains(t, mockSMS.lastBody, "regular bin collection day")
	assert.Contains(t, mockSMS.lastBody, "no collections")
	assert.True(t, result.SMSSent)
	assert.Equal(t, 0, len(result.Collections))
}

func TestNotifier_NoSmsWhenNoCollectionsAndNotRegularDay(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday, but regular day is Wednesday
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)                     // Collections are next week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.RegularCollectionDay = int(time.Wednesday) // Regular day is Wednesday, not Tuesday
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.False(t, mockSMS.sendSmsCalled)
	assert.False(t, result.SMSSent)
	assert.Equal(t, 0, len(result.Collections))
}

func TestNotifier_ScraperErrorPropagated(t *testing.T) {
	expectedError := errors.New("scraper failed")
	mockScr := &mockScraper{err: expectedError}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return time.Now() },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)
	assert.False(t, mockSMS.sendSmsCalled)
}

func TestNotifier_SmsErrorPropagated(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	expectedError := errors.New("SMS send failed")
	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{err: expectedError}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)
	assert.True(t, mockSMS.sendSmsCalled)
}

func TestNotifier_UsesTodayDateFromConfig(t *testing.T) {
	// Clock returns a different date, but config overrides it
	clock := func() time.Time { return time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) }

	// Config says today is Jan 15, so tomorrow is Jan 16
	tomorrow := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     clock,
	}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15" // Override the clock
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockSMS.sendSmsCalled)
	assert.Contains(t, mockSMS.lastBody, "General Waste")
	assert.True(t, result.SMSSent)
}

func TestNotifier_InvalidTodayDateReturnsError(t *testing.T) {
	mockScr := &mockScraper{binTimes: []scraper.BinTime{}}
	mockSMS := &mockSMSClient{}

	notifier := &Notifier{
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return time.Now() },
	}

	cfg := createTestConfig()
	cfg.TodayDate = "invalid-date"
	result, err := notifier.Run(cfg)

	assert.NotNil(t, err)
	assert.Nil(t, result)
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
		Scraper:   mockScr,
		SMSClient: mockSMS,
		Clock:     func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.DryRun = true
	_, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockSMS.sendSmsCalled)
	assert.True(t, mockSMS.lastDryRun)
}
