package helper

import (
	"strings"
	"testing"
)

func TestCreateInputDatabaseSQLSupportsSchemaConstraintsAndMultiRowInsert(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	input := `
		CREATE TABLE users (
			tenant_id INTEGER NOT NULL,
			id INTEGER NOT NULL,
			name TEXT UNIQUE,
			PRIMARY KEY (tenant_id, id)
		);
		CREATE TABLE posts (
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			UNIQUE (tenant_id, title),
			FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id)
		);
		INSERT INTO users (tenant_id, id, name) VALUES
			(1, 1, 'Ada'),
			(1, 2, 'Linus');
		INSERT INTO posts (tenant_id, user_id, title) VALUES
			(1, 2, 'Hello');
	`

	if err := CreateInputDatabase(db, input, InputFormatSQL); err != nil {
		t.Fatalf("CreateInputDatabase returned error: %v", err)
	}

	var name string
	if err := db.QueryRow(`
		SELECT users.name
		FROM posts
		JOIN users ON users.tenant_id = posts.tenant_id AND users.id = posts.user_id
	`).Scan(&name); err != nil {
		t.Fatalf("query initialized database: %v", err)
	}
	if name != "Linus" {
		t.Fatalf("name = %q, want Linus", name)
	}

	rows, err := db.Query(`PRAGMA foreign_key_list('posts')`)
	if err != nil {
		t.Fatalf("inspect foreign key: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if count != 2 {
		t.Fatalf("foreign key column count = %d, want 2", count)
	}
}

func TestCreateInputDatabaseSQLRollsBackInvalidInput(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	err = CreateInputDatabase(db, `
		CREATE TABLE users (id INTEGER PRIMARY KEY);
		INSERT INTO missing (id) VALUES (1);
	`, InputFormatSQL)
	if err == nil || !strings.Contains(err.Error(), "input statement 2 failed") {
		t.Fatalf("error = %v, want statement context", err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = 'users'`).Scan(&count); err != nil {
		t.Fatalf("inspect rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("table count = %d, want 0 after rollback", count)
	}
}

func TestCreateInputDatabaseSQLEnforcesCompositeForeignKey(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	err = CreateInputDatabase(db, `
		CREATE TABLE users (
			tenant_id INTEGER,
			id INTEGER,
			PRIMARY KEY (tenant_id, id)
		);
		CREATE TABLE posts (
			tenant_id INTEGER,
			user_id INTEGER,
			FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id)
		);
		INSERT INTO posts (tenant_id, user_id) VALUES (1, 99);
	`, InputFormatSQL)
	if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		t.Fatalf("error = %v, want composite foreign key failure", err)
	}
}

func TestCreateInputDatabaseDefaultsToCSV(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	if err := CreateInputDatabase(db, "id,name\n1,Ada\n", ""); err != nil {
		t.Fatalf("CreateInputDatabase returned error: %v", err)
	}

	var name string
	if err := db.QueryRow(`SELECT name FROM input`).Scan(&name); err != nil {
		t.Fatalf("query CSV input: %v", err)
	}
	if name != "Ada" {
		t.Fatalf("name = %q, want Ada", name)
	}
}
