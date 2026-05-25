package helper

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
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

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.Remove(dbPath)
		return nil, nil, err
	}

	cleanup := func() {
		os.Remove(dbPath)
	}

	return db, cleanup, nil
}
