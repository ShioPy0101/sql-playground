package helper

import (
	"reflect"
	"testing"
)

func TestPreviewSQLInputTables(t *testing.T) {
	tables, err := PreviewInputTables(`
		CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);
		CREATE TABLE "empty ""table" (id INTEGER, body TEXT);
		INSERT INTO users VALUES (1, '坂本龍馬'), (2, 'comma, and
newline'), (3, NULL);
	`, InputFormatSQL)
	if err != nil {
		t.Fatal(err)
	}
	want := []InputTable{
		{Name: "users", Rows: [][]string{{"id", "name"}, {"1", "坂本龍馬"}, {"2", "comma, and\nnewline"}, {"3", ""}}},
		{Name: `empty "table`, Rows: [][]string{{"id", "body"}}},
	}
	if !reflect.DeepEqual(tables, want) {
		t.Fatalf("tables = %#v, want %#v", tables, want)
	}
}

func TestPreviewSQLInputRejectsInvalidSetup(t *testing.T) {
	if _, err := PreviewInputTables("CREATE TABLE users (id INTEGER); INSERT INTO missing VALUES (1);", InputFormatSQL); err == nil {
		t.Fatal("expected initialization error")
	}
}
