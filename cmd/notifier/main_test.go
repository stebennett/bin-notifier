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

// mockNotificationClient is a mock implementation of NotificationClient for testing
type mockNotificationClient struct {
	sendCalled bool
	lastURL    string
	lastBody   string
	lastTag    string
	lastDryRun bool
	err        error
}

func (m *mockNotificationClient) SendNotification(url string, body string, tag string, dryRun bool) error {
	m.sendCalled = true
	m.lastURL = url
	m.lastBody = body
	m.lastTag = tag
	m.lastDryRun = dryRun
	return m.err
}

func createTestConfig() config.Config {
	return config.Config{
		PostCode:             "RG12 1AB",
		AddressCode:          "12345",
		RegularCollectionDay: int(time.Tuesday),
		AppriseURL:           "http://apprise:8000/notify/",
		DryRun:               false,
	}
}

func TestNotifier_SendsNotificationWhenCollectionTomorrow(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	tomorrow := today.AddDate(0, 0, 1)                     // Tuesday

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
			{Type: "Recycling", CollectionTime: tomorrow},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.Equal(t, cfg.AppriseURL, mockNotify.lastURL)
	assert.Contains(t, mockNotify.lastBody, "General Waste")
	assert.Contains(t, mockNotify.lastBody, "Recycling")
	assert.True(t, result.SMSSent)
	assert.Equal(t, 2, len(result.Collections))
}

func TestNotifier_SendsNotificationOnRegularDayNoCollections(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday (regular collection day), but no collections
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)                     // Collections are next week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.RegularCollectionDay = int(time.Tuesday) // Tomorrow is Tuesday
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.Contains(t, mockNotify.lastBody, "regular bin collection day")
	assert.Contains(t, mockNotify.lastBody, "no collections")
	assert.True(t, result.SMSSent)
	assert.Equal(t, 0, len(result.Collections))
}

func TestNotifier_NoNotificationWhenNoCollectionsAndNotRegularDay(t *testing.T) {
	// Set up: today is Monday, tomorrow is Tuesday, but regular day is Wednesday
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday
	nextWeek := today.AddDate(0, 0, 7)                     // Collections are next week

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: nextWeek},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.RegularCollectionDay = int(time.Wednesday) // Regular day is Wednesday, not Tuesday
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.False(t, mockNotify.sendCalled)
	assert.False(t, result.SMSSent)
	assert.Equal(t, 0, len(result.Collections))
}

func TestNotifier_ScraperErrorPropagated(t *testing.T) {
	expectedError := errors.New("scraper failed")
	mockScr := &mockScraper{err: expectedError}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return time.Now() },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)
	assert.False(t, mockNotify.sendCalled)
}

func TestNotifier_NotificationErrorPropagated(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	expectedError := errors.New("notification send failed")
	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockNotify := &mockNotificationClient{err: expectedError}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	result, err := notifier.Run(cfg)

	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)
	assert.True(t, mockNotify.sendCalled)
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
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              clock,
	}

	cfg := createTestConfig()
	cfg.TodayDate = "2024-01-15" // Override the clock
	result, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.Contains(t, mockNotify.lastBody, "General Waste")
	assert.True(t, result.SMSSent)
}

func TestNotifier_InvalidTodayDateReturnsError(t *testing.T) {
	mockScr := &mockScraper{binTimes: []scraper.BinTime{}}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return time.Now() },
	}

	cfg := createTestConfig()
	cfg.TodayDate = "invalid-date"
	result, err := notifier.Run(cfg)

	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestNotifier_DryRunPassedToNotificationClient(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.DryRun = true
	_, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.True(t, mockNotify.lastDryRun)
}

func TestNotifier_TagPassedToNotificationClient(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	cfg.AppriseTag = "sms"
	_, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.Equal(t, "sms", mockNotify.lastTag)
}

func TestNotifier_EmptyTagPassedWhenNotConfigured(t *testing.T) {
	today := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	mockScr := &mockScraper{
		binTimes: []scraper.BinTime{
			{Type: "General Waste", CollectionTime: tomorrow},
		},
	}
	mockNotify := &mockNotificationClient{}

	notifier := &Notifier{
		Scraper:            mockScr,
		NotificationClient: mockNotify,
		Clock:              func() time.Time { return today },
	}

	cfg := createTestConfig()
	// AppriseTag not set, should be empty string
	_, err := notifier.Run(cfg)

	assert.Nil(t, err)
	assert.True(t, mockNotify.sendCalled)
	assert.Equal(t, "", mockNotify.lastTag)
}
