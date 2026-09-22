package helper

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

// OpenTempSQLiteDB creates an isolated database file for a single execution.
func OpenTempSQLiteDB() (*sql.DB, func(), error) {
	db, _, cleanup, err := OpenTempSQLiteDBWithPath()
	return db, cleanup, err
}

// OpenTempSQLiteDBWithPath also exposes the temporary file path for helpers
// that need an independent SQLite connection for native statement metrics.
func OpenTempSQLiteDBWithPath() (*sql.DB, string, func(), error) {
	tmpFile, err := os.CreateTemp("", "sqlite-playground-*.sqlite")
	if err != nil {
		return nil, "", nil, err
	}

	dbPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		os.Remove(dbPath)
		return nil, "", nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		os.Remove(dbPath)
		return nil, "", nil, err
	}
	// Foreign-key enforcement is connection-local in SQLite. A single
	// connection keeps the setting stable for the lifetime of this temporary DB.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		os.Remove(dbPath)
		return nil, "", nil, err
	}

	cleanup := func() {
		os.Remove(dbPath)
	}

	return db, dbPath, cleanup, nil
}
