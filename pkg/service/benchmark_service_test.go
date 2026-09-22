package service

import (
	"strings"
	"testing"
)

func TestBenchmarkServiceRunsAllConfiguredStages(t *testing.T) {
	service := NewBenchmarkService()
	report, err := service.Run("sql", BenchmarkConfig{
		Enabled:   true,
		Target:    "submission",
		RowCounts: []int{1_000, 10_000, 50_000, 100_000},
	}, `SELECT * FROM posts WHERE tenant_id = 42;`)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != 4 {
		t.Fatalf("report = %#v, want four completed stages", report)
	}

	previousSteps := int64(0)
	for _, result := range report.Results {
		if result.VMSteps <= previousSteps {
			t.Fatalf("rowCount %d VM_STEP = %d, want greater than %d", result.RowCount, result.VMSteps, previousSteps)
		}
		if result.FullScanSteps != int64(result.RowCount-1) {
			t.Fatalf("rowCount %d FULLSCAN_STEP = %d, want %d", result.RowCount, result.FullScanSteps, result.RowCount-1)
		}
		if result.DatabaseSizeBytes <= 0 || result.DataGenerationMS <= 0 || result.ExecutionTimeMS < 0 {
			t.Fatalf("rowCount %d diagnostics are incomplete: %#v", result.RowCount, result)
		}
		if !strings.Contains(strings.Join(result.QueryPlan, "\n"), "SCAN posts") {
			t.Fatalf("rowCount %d plan = %#v, want SCAN posts", result.RowCount, result.QueryPlan)
		}
		previousSteps = result.VMSteps
		t.Logf(
			"rows=%d size=%d generation=%.2fms select=%.2fms VM_STEP=%d FULLSCAN_STEP=%d",
			result.RowCount,
			result.DatabaseSizeBytes,
			result.DataGenerationMS,
			result.ExecutionTimeMS,
			result.VMSteps,
			result.FullScanSteps,
		)
	}
}

func TestBenchmarkServiceAppliesDDLBeforeFixedQuery(t *testing.T) {
	report, err := NewBenchmarkService().Run("ddl", BenchmarkConfig{
		Enabled:   true,
		Target:    "fixed-query",
		Query:     `SELECT * FROM posts WHERE tenant_id = 42;`,
		RowCounts: []int{1_000},
	}, `
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			body TEXT NOT NULL
		);
		CREATE INDEX idx_posts_tenant_id ON posts(tenant_id);
	`)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != 1 {
		t.Fatalf("report = %#v, want one completed stage", report)
	}
	result := report.Results[0]
	if result.FullScanSteps != 0 {
		t.Fatalf("FULLSCAN_STEP = %d, want 0 with index", result.FullScanSteps)
	}
	if !strings.Contains(strings.Join(result.QueryPlan, "\n"), "USING INDEX idx_posts_tenant_id") {
		t.Fatalf("plan = %#v, want learner index", result.QueryPlan)
	}
}

func TestBenchmarkServiceStopsAfterStageFailure(t *testing.T) {
	report, err := NewBenchmarkService().Run("ddl", BenchmarkConfig{
		Enabled:   true,
		Target:    "fixed-query",
		Query:     `SELECT * FROM posts WHERE tenant_id = 42;`,
		RowCounts: []int{1_000, 10_000},
	}, `CREATE TABLE wrong_table (id INTEGER PRIMARY KEY);`)
	if err != nil {
		t.Fatalf("Run returned configuration error: %v", err)
	}
	if report.Status != "failed" || len(report.Results) != 0 {
		t.Fatalf("report = %#v, want failure before later stages", report)
	}
	if !strings.Contains(report.StoppedReason, "1000行") {
		t.Fatalf("stopped reason = %q, want first stage", report.StoppedReason)
	}
}

func TestBenchmarkServiceRejectsUnsafeSQLTarget(t *testing.T) {
	_, err := NewBenchmarkService().Run("sql", BenchmarkConfig{
		Enabled:   true,
		Target:    "submission",
		RowCounts: []int{1_000},
	}, `DELETE FROM posts;`)
	if err == nil {
		t.Fatal("Run accepted non-SELECT submission")
	}
}
