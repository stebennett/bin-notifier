package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	var name string
	require.NoError(t, s.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='collections'`).Scan(&name))
	require.Equal(t, "collections", name)

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
