package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerExecutesSQL(t *testing.T) {
	body := []byte(`{
		"csv": "name,score\nAlice,82\nBob,91",
		"query": "SELECT name FROM input ORDER BY score DESC;"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/sqlite/execute", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res executeResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := "name\nBob\nAlice\n"
	if res.CSV != want {
		t.Fatalf("expected csv %q, got %q", want, res.CSV)
	}
	if len(res.Metrics.Statements) != 1 {
		t.Fatalf("expected one statement metric, got %d", len(res.Metrics.Statements))
	}
	if len(res.Metrics.Statements[0].QueryPlan) == 0 {
		t.Fatal("expected a query plan for SELECT")
	}
}

func TestHandlerExecutesSQLWithInspection(t *testing.T) {
	body := []byte(`{
		"csv": "",
		"query": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);",
		"checkSql": "SELECT name, type, \"notnull\", pk FROM pragma_table_info('users') ORDER BY cid;"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/sqlite/execute", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res executeResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := "name,type,notnull,pk\nid,INTEGER,0,1\nname,TEXT,1,0\n"
	if res.CSV != want {
		t.Fatalf("expected csv %q, got %q", want, res.CSV)
	}
}

func TestHandlerExecutesQueryAgainstSQLInput(t *testing.T) {
	body := []byte(`{
		"inputType": "sql",
		"input": "CREATE TABLE users (tenant_id INTEGER, id INTEGER, name TEXT, PRIMARY KEY (tenant_id, id)); INSERT INTO users VALUES (1, 1, 'Ada'), (1, 2, 'Linus');",
		"query": "SELECT name FROM users ORDER BY id DESC;"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/sqlite/execute", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res executeResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := "name\nLinus\nAda\n"
	if res.CSV != want {
		t.Fatalf("expected csv %q, got %q", want, res.CSV)
	}
}

func TestHandlerRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sqlite/execute", nil)
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
