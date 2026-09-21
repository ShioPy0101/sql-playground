package service

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSubmissionStoreMigratesLegacyTableWithNullableEventID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE task_submissions (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL, task_number INTEGER NOT NULL, task_slug TEXT NOT NULL, task_title TEXT NOT NULL, query TEXT NOT NULL, passed INTEGER NOT NULL, cases_json TEXT NOT NULL, submitted_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewSubmissionStore(path)
	if err != nil {
		t.Fatalf("NewSubmissionStore returned error: %v", err)
	}
	defer store.db.Close()
	if !store.sqliteColumnExists("task_submissions", "event_id") {
		t.Fatalf("event_id column was not added")
	}
}
