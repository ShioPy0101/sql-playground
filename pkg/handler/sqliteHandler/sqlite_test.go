package sqliteHandler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShioPy0101/sql-playground/pkg/service"
	"github.com/labstack/echo/v4"
)

func TestExecuteKeepsCSVResponseAndAddsMetrics(t *testing.T) {
	handler := NewSQLiteHandler(service.NewSQLiteService())
	e := echo.New()
	body := []byte(`{"csv":"id,name\n1,Ada\n2,Linus","query":"SELECT name FROM input ORDER BY id"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sqlite/execute", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	if err := handler.Execute(e.NewContext(req, rec)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response ExecuteResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.CSV != "name\nAda\nLinus\n" {
		t.Fatalf("CSV = %q, want existing response unchanged", response.CSV)
	}
	if len(response.Metrics.Statements) != 1 {
		t.Fatalf("statement metrics length = %d, want 1", len(response.Metrics.Statements))
	}
}
