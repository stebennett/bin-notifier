package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS collections (
    location    TEXT NOT NULL,
    bin_type    TEXT NOT NULL,
    date        TEXT NOT NULL,
    scraped_at  TEXT NOT NULL,
    PRIMARY KEY (location, bin_type, date)
);
CREATE INDEX IF NOT EXISTS idx_collections_location_date ON collections(location, date);
CREATE TABLE IF NOT EXISTS seen_locations (
    location TEXT PRIMARY KEY
);
`

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the DB is reachable; used by /healthz.
func (s *Store) Ping() error { return s.db.Ping() }

// Collection represents a single bin collection event.
type Collection struct {
	BinType string `json:"bin_type"`
	Date    string `json:"date"` // YYYY-MM-DD
}

// ErrNoData is returned when a location has never been written to the store.
var ErrNoData = errors.New("no data for location")

// ReplaceCollections atomically replaces all collection rows for location with items,
// recording scrapedAt as the scrape timestamp. Passing nil/empty items clears the location.
func (s *Store) ReplaceCollections(location string, scrapedAt time.Time, items []Collection) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT OR IGNORE INTO seen_locations(location) VALUES (?)`, location); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM collections WHERE location = ?`, location); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO collections (location, bin_type, date, scraped_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scrapedStr := scrapedAt.UTC().Format(time.RFC3339)
	for _, c := range items {
		if _, err := stmt.Exec(location, c.BinType, c.Date, scrapedStr); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListCollections returns rows for location with date >= from, optionally filtered by bin types.
// Returns ErrNoData if the location has never been written to (no ReplaceCollections call yet).
func (s *Store) ListCollections(location string, from string, types []string) ([]Collection, time.Time, error) {
	var seen bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM seen_locations WHERE location = ?)`, location).Scan(&seen); err != nil {
		return nil, time.Time{}, err
	}
	if !seen {
		return nil, time.Time{}, ErrNoData
	}

	args := []any{location, from}
	q := `SELECT bin_type, date, scraped_at FROM collections WHERE location = ? AND date >= ?`
	if len(types) > 0 {
		q += ` AND bin_type IN (?` + repeatPlaceholders(len(types)-1) + `)`
		for _, t := range types {
			args = append(args, t)
		}
	}
	q += ` ORDER BY date ASC, bin_type ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()

	var out []Collection
	var latestScraped time.Time
	for rows.Next() {
		var c Collection
		var scrapedStr string
		if err := rows.Scan(&c.BinType, &c.Date, &scrapedStr); err != nil {
			return nil, time.Time{}, err
		}
		ts, err := time.Parse(time.RFC3339, scrapedStr)
		if err != nil {
			return nil, time.Time{}, err
		}
		if ts.After(latestScraped) {
			latestScraped = ts
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, err
	}
	if latestScraped.IsZero() {
		// Location seen but no rows match the filters / from — fetch any scraped_at for context.
		var scrapedStr string
		if err := s.db.QueryRow(`SELECT scraped_at FROM collections WHERE location = ? LIMIT 1`, location).Scan(&scrapedStr); err == nil {
			if ts, perr := time.Parse(time.RFC3339, scrapedStr); perr == nil {
				latestScraped = ts
			}
		}
	}
	return out, latestScraped, nil
}

func repeatPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += ", ?"
	}
	return s
}
