package store

import (
	"database/sql"
	"fmt"

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
