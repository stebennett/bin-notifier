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
    PRIMARY KEY (location, bin_type, date)
);
CREATE INDEX IF NOT EXISTS idx_collections_location_date ON collections(location, date);
CREATE TABLE IF NOT EXISTS seen_locations (
    location   TEXT PRIMARY KEY,
    scraped_at TEXT NOT NULL
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

// ErrNoData is returned when a location has never been registered with the store
// (i.e. has never been the target of a ReplaceCollections call). A location that has
// been registered but currently holds zero rows (e.g. cleared via ReplaceCollections
// with empty items) does not return ErrNoData.
var ErrNoData = errors.New("no data for location")

// ReplaceCollections atomically replaces all collection rows for location with items,
// recording scrapedAt as the scrape timestamp. Passing nil/empty items clears the location.
func (s *Store) ReplaceCollections(location string, scrapedAt time.Time, items []Collection) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	scrapedStr := scrapedAt.UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`INSERT INTO seen_locations(location, scraped_at) VALUES (?, ?)
		 ON CONFLICT(location) DO UPDATE SET scraped_at = excluded.scraped_at`,
		location, scrapedStr,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM collections WHERE location = ?`, location); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO collections (location, bin_type, date) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range items {
		if _, err := stmt.Exec(location, c.BinType, c.Date); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListCollections returns rows for location with date >= from, optionally filtered by bin types.
// Rows are returned ordered by date ASC, bin_type ASC — NextCollection depends on this contract.
// Returns ErrNoData if the location has never been written to (no ReplaceCollections call yet).
func (s *Store) ListCollections(location string, from string, types []string) ([]Collection, time.Time, error) {
	var scrapedStr string
	err := s.db.QueryRow(`SELECT scraped_at FROM seen_locations WHERE location = ?`, location).Scan(&scrapedStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, ErrNoData
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	scrapedAt, err := time.Parse(time.RFC3339, scrapedStr)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parse scraped_at for %q: %w", location, err)
	}

	args := []any{location, from}
	q := `SELECT bin_type, date FROM collections WHERE location = ? AND date >= ?`
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
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.BinType, &c.Date); err != nil {
			return nil, time.Time{}, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, err
	}
	return out, scrapedAt, nil
}

// ErrNoMatch is returned by NextCollection when the location is known but no
// future collection matches the supplied filters.
var ErrNoMatch = errors.New("no matching collection")

// NextCollection returns the earliest upcoming collection date for location (>= from),
// optionally filtered by bin types, along with all bin types collected on that date and
// the scrape timestamp. Returns ErrNoData if location was never registered, or
// ErrNoMatch if no matching future row exists.
func (s *Store) NextCollection(location string, from string, types []string) (string, []string, time.Time, error) {
	rows, scrapedAt, err := s.ListCollections(location, from, types)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if len(rows) == 0 {
		return "", nil, time.Time{}, ErrNoMatch
	}
	earliest := rows[0].Date
	var binTypes []string
	for _, r := range rows {
		if r.Date != earliest {
			break
		}
		binTypes = append(binTypes, r.BinType)
	}
	return earliest, binTypes, scrapedAt, nil
}

func repeatPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += ", ?"
	}
	return s
}
