package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stebennett/bin-notifier/pkg/scraper"
	"github.com/stretchr/testify/assert"
)

func TestSetAndGet(t *testing.T) {
	c := New(6 * time.Hour)

	bins := []scraper.BinTime{
		{Type: "Recycling", CollectionTime: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)},
	}

	c.Set("RG12 1AB", "12345", bins)

	got, ok := c.Get("RG12 1AB", "12345")
	assert.True(t, ok)
	assert.Equal(t, bins, got)
}

func TestGetMissing(t *testing.T) {
	c := New(6 * time.Hour)

	got, ok := c.Get("RG12 1AB", "12345")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestExpiry(t *testing.T) {
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	c := New(1 * time.Hour)
	c.now = func() time.Time { return now }

	bins := []scraper.BinTime{
		{Type: "Recycling", CollectionTime: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)},
	}

	c.Set("RG12 1AB", "12345", bins)

	// Still valid
	got, ok := c.Get("RG12 1AB", "12345")
	assert.True(t, ok)
	assert.Equal(t, bins, got)

	// Advance past TTL
	c.now = func() time.Time { return now.Add(2 * time.Hour) }
	got, ok = c.Get("RG12 1AB", "12345")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestInvalidate(t *testing.T) {
	c := New(6 * time.Hour)

	bins := []scraper.BinTime{
		{Type: "Recycling", CollectionTime: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)},
	}

	c.Set("RG12 1AB", "12345", bins)
	c.Invalidate("RG12 1AB", "12345")

	got, ok := c.Get("RG12 1AB", "12345")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestConcurrentAccess(t *testing.T) {
	c := New(6 * time.Hour)

	bins := []scraper.BinTime{
		{Type: "Recycling", CollectionTime: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Set("RG12 1AB", "12345", bins)
		}()
		go func() {
			defer wg.Done()
			c.Get("RG12 1AB", "12345")
		}()
	}
	wg.Wait()

	got, ok := c.Get("RG12 1AB", "12345")
	assert.True(t, ok)
	assert.Equal(t, bins, got)
}
