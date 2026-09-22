package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShioPy0101/sql-playground/pkg/service"
)

func TestHandlerRunsConfiguredBenchmark(t *testing.T) {
	taskDir := t.TempDir()
	writeBenchmarkTask(t, taskDir, "901.json", `{
		"number": 901,
		"mode": "sql",
		"benchmark": {
			"enabled": true,
			"target": "submission",
			"rowCounts": [1000]
		}
	}`)
	t.Setenv("TASKS_DIR", taskDir)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/901/benchmark", bytes.NewBufferString(
		`{"query":"SELECT * FROM posts WHERE tenant_id = 42;"}`,
	))
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var report service.BenchmarkReport
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != 1 {
		t.Fatalf("report = %#v, want one completed result", report)
	}
	if report.Results[0].VMSteps <= 0 || report.Results[0].FullScanSteps != 999 {
		t.Fatalf("native counters = %#v", report.Results[0])
	}
}

func TestHandlerDoesNotRunWithoutBenchmarkConfiguration(t *testing.T) {
	taskDir := t.TempDir()
	writeBenchmarkTask(t, taskDir, "902.json", `{"number": 902, "mode": "sql"}`)
	t.Setenv("TASKS_DIR", taskDir)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/902/benchmark", bytes.NewBufferString(
		`{"query":"SELECT * FROM posts WHERE tenant_id = 42;"}`,
	))
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func writeBenchmarkTask(t *testing.T, dir string, name string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write task: %v", err)
	}
}
