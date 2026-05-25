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
}

func TestHandlerRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sqlite/execute", nil)
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
