package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	for _, table := range []string{"collections", "seen_locations"} {
		var name string
		require.NoError(t, s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name))
		require.Equal(t, table, name)
	}

	// Re-opening an existing file must succeed (schema migration is idempotent).
	s2, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s2.Close())
}

func TestOpenInMemory(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, s.Close())
}

func TestReplaceCollectionsInsertsRows(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	err = s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	})
	require.NoError(t, err)

	got, gotScrapedAt, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, scrapedAt.UTC(), gotScrapedAt.UTC())
	require.Equal(t, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
		{BinType: "Food Waste", Date: "2026-05-08"},
	}, got)
}

func TestReplaceCollectionsReplacesExistingLocation(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "Food Waste", Date: "2026-05-08"},
	}))

	got, _, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, []Collection{
		{BinType: "Food Waste", Date: "2026-05-08"},
	}, got)
}

func TestReplaceCollectionsEmptyClearsLocation(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, nil))

	got, _, err := s.ListCollections("Home", "2026-05-05", nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestReplaceCollectionsDoesNotAffectOtherLocations(t *testing.T) {
	s, err := Open(":memory:")
	require.NoError(t, err)
	defer s.Close()

	scrapedAt := time.Date(2026, 5, 5, 18, 0, 0, 0, time.UTC)
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, []Collection{
		{BinType: "General Waste", Date: "2026-05-07"},
	}))
	require.NoError(t, s.ReplaceCollections("Office", scrapedAt, []Collection{
		{BinType: "Recycling", Date: "2026-05-09"},
	}))
	require.NoError(t, s.ReplaceCollections("Home", scrapedAt, nil))

	got, _, err := s.ListCollections("Office", "2026-05-05", nil)
	require.NoError(t, err)
	require.Equal(t, []Collection{{BinType: "Recycling", Date: "2026-05-09"}}, got)
}
