package helper

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

// OpenTempSQLiteDB creates an isolated database file for a single execution.
func OpenTempSQLiteDB() (*sql.DB, func(), error) {
	tmpFile, err := os.CreateTemp("", "sqlite-playground-*.sqlite")
	if err != nil {
		return nil, nil, err
	}

	dbPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		os.Remove(dbPath)
		return nil, nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		os.Remove(dbPath)
		return nil, nil, err
	}

	cleanup := func() {
		os.Remove(dbPath)
	}

	return db, cleanup, nil
}
