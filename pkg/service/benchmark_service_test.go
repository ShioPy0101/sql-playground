package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ShioPy0101/sql-playground/pkg/service/helper"
)

func TestBenchmarkServiceRunsAllConfiguredStages(t *testing.T) {
	service := NewBenchmarkService()
	report, err := service.Run("sql", BenchmarkConfig{
		Enabled:   true,
		Target:    "submission",
		RowCounts: []int{1_000, 10_000, 50_000, 100_000},
		SchemaSQL: benchmarkPostsSchema(),
		Dataset:   benchmarkPostsDataset(),
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
		Dataset:   benchmarkPostsDataset(),
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

func TestBenchmarkServiceMeasuresOrdersIndexTask(t *testing.T) {
	report, err := NewBenchmarkService().Run("ddl", BenchmarkConfig{
		Enabled:   true,
		Target:    "fixed-query",
		Query:     `SELECT * FROM orders WHERE customer_id = 42;`,
		RowCounts: []int{1_000},
		Dataset:   benchmarkOrdersDataset(),
	}, `
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			ordered_at TEXT NOT NULL,
			status TEXT NOT NULL,
			total INTEGER NOT NULL
		);
		CREATE INDEX idx_orders_customer_id ON orders(customer_id);
	`)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != 1 {
		t.Fatalf("report = %#v, want one completed stage", report)
	}
	result := report.Results[0]
	if result.FullScanSteps != 0 {
		t.Fatalf("FULLSCAN_STEP = %d, want 0 with orders index", result.FullScanSteps)
	}
	if !strings.Contains(strings.Join(result.QueryPlan, "\n"), "USING INDEX idx_orders_customer_id") {
		t.Fatalf("plan = %#v, want orders customer index", result.QueryPlan)
	}
}

func TestTask45BenchmarkConfigurationRunsAllStages(t *testing.T) {
	task, err := LoadTaskDefinition("45")
	if err != nil {
		t.Fatalf("LoadTaskDefinition returned error: %v", err)
	}
	if task.Mode != "ddl" || task.Benchmark == nil || !task.Benchmark.Enabled || !task.Benchmark.RunOnFailed {
		t.Fatalf("task 45 benchmark configuration = %#v", task.Benchmark)
	}

	report, err := NewBenchmarkService().Run(task.Mode, *task.Benchmark, task.SolutionSQL)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != len(task.Benchmark.RowCounts) {
		t.Fatalf("report = %#v, want all configured stages", report)
	}
	for _, result := range report.Results {
		if result.FullScanSteps != 0 {
			t.Fatalf("rowCount %d FULLSCAN_STEP = %d, want 0", result.RowCount, result.FullScanSteps)
		}
		if !strings.Contains(strings.Join(result.QueryPlan, "\n"), "USING INDEX idx_orders_customer_id") {
			t.Fatalf("rowCount %d plan = %#v, want customer index", result.RowCount, result.QueryPlan)
		}
	}
}

func TestTask45BenchmarkWithoutIndexesShowsFullScan(t *testing.T) {
	task, err := LoadTaskDefinition("45")
	if err != nil {
		t.Fatalf("LoadTaskDefinition returned error: %v", err)
	}
	config := *task.Benchmark
	config.RowCounts = []int{1_000}

	report, err := NewBenchmarkService().Run(task.Mode, config, `
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			ordered_at TEXT NOT NULL,
			status TEXT NOT NULL,
			total INTEGER NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Status != "completed" || len(report.Results) != 1 {
		t.Fatalf("report = %#v, want one completed stage", report)
	}
	result := report.Results[0]
	if result.FullScanSteps != 999 {
		t.Fatalf("FULLSCAN_STEP = %d, want 999 without index", result.FullScanSteps)
	}
	if !strings.Contains(strings.Join(result.QueryPlan, "\n"), "SCAN orders") {
		t.Fatalf("plan = %#v, want full table scan", result.QueryPlan)
	}
}

func TestSaaSCurriculumBenchmarksRunAllStages(t *testing.T) {
	reports := make(map[int]BenchmarkReport)
	benchmarkTaskNumbers := []int{48, 49, 50, 51, 52, 53, 54, 55, 56, 58}
	for _, number := range benchmarkTaskNumbers {
		task, err := LoadTaskDefinition(fmt.Sprintf("%d", number))
		if err != nil {
			t.Fatalf("LoadTaskDefinition(%d) returned error: %v", number, err)
		}
		if task.Benchmark == nil || !task.Benchmark.Enabled {
			t.Fatalf("task %d has no enabled benchmark", number)
		}
		report, err := NewBenchmarkService().Run(task.Mode, *task.Benchmark, task.SolutionSQL)
		if err != nil {
			t.Fatalf("task %d benchmark returned error: %v", number, err)
		}
		if report.Status != "completed" || len(report.Results) != 4 {
			t.Fatalf("task %d report = %#v, want four completed stages", number, report)
		}
		reports[number] = report
	}

	withoutIndex := reports[51].Results[3]
	withIndex := reports[56].Results[3]
	if withoutIndex.FullScanSteps == 0 || withIndex.FullScanSteps != 0 {
		t.Fatalf("tenant search FULLSCAN before=%d after=%d", withoutIndex.FullScanSteps, withIndex.FullScanSteps)
	}
	if !strings.Contains(strings.Join(withIndex.QueryPlan, "\n"), "USING INDEX idx_posts_tenant_id") {
		t.Fatalf("task 56 plan = %#v, want tenant index", withIndex.QueryPlan)
	}
	composite := reports[58].Results[3]
	if composite.SortCount != 0 || !strings.Contains(strings.Join(composite.QueryPlan, "\n"), "idx_posts_tenant_created_at") {
		t.Fatalf("task 58 metrics = %#v, want composite index without sort", composite)
	}
}

func TestBenchmarkServiceStopsAtConfiguredTimeout(t *testing.T) {
	task, err := LoadTaskDefinition("45")
	if err != nil {
		t.Fatalf("LoadTaskDefinition returned error: %v", err)
	}
	config := *task.Benchmark
	config.RowCounts = []int{100_000}
	config.TimeoutMS = 1

	report, err := NewBenchmarkService().Run(task.Mode, config, task.SolutionSQL)
	if err != nil {
		t.Fatalf("Run returned configuration error: %v", err)
	}
	if report.Status != "failed" || len(report.Results) != 0 {
		t.Fatalf("report = %#v, want timeout before a result", report)
	}
	if !strings.Contains(report.StoppedReason, "context deadline exceeded") {
		t.Fatalf("stopped reason = %q, want timeout", report.StoppedReason)
	}
}

func TestBenchmarkServiceStopsAfterStageFailure(t *testing.T) {
	report, err := NewBenchmarkService().Run("ddl", BenchmarkConfig{
		Enabled:   true,
		Target:    "fixed-query",
		Query:     `SELECT * FROM posts WHERE tenant_id = 42;`,
		RowCounts: []int{1_000, 10_000},
		Dataset:   benchmarkPostsDataset(),
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

func benchmarkPostsSchema() string {
	return `CREATE TABLE posts (
		id INTEGER PRIMARY KEY,
		tenant_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		body TEXT NOT NULL
	);`
}

func benchmarkPostsDataset() *BenchmarkDataset {
	return &BenchmarkDataset{
		Table: "posts",
		Columns: []BenchmarkColumn{
			{Name: "id", Type: "INTEGER", Expression: "row_number"},
			{Name: "tenant_id", Type: "INTEGER", Expression: "(row_number % 100) + 1"},
			{Name: "user_id", Type: "INTEGER", Expression: "(row_number % 50000) + 1"},
			{Name: "body", Type: "TEXT", Expression: "'benchmark body'"},
		},
	}
}

func benchmarkOrdersDataset() *BenchmarkDataset {
	return &BenchmarkDataset{
		Table: "orders",
		Columns: []BenchmarkColumn{
			{Name: "id", Type: "INTEGER", Expression: "row_number"},
			{Name: "customer_id", Type: "INTEGER", Expression: "(row_number % 100) + 1"},
			{Name: "ordered_at", Type: "TEXT", Expression: "printf('2026-%02d-%02d', ((row_number - 1) % 12) + 1, ((row_number - 1) % 28) + 1)"},
			{Name: "status", Type: "TEXT", Expression: "CASE WHEN row_number % 4 = 0 THEN 'canceled' ELSE 'paid' END"},
			{Name: "total", Type: "INTEGER", Expression: "(row_number % 5000) + 100"},
		},
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

func TestBenchmarkTimeoutUsesTaskValueAndServerMaximum(t *testing.T) {
	t.Setenv("SQLITE_BENCHMARK_TIMEOUT_MS", "2000")

	timeout, err := benchmarkTimeout(750)
	if err != nil || timeout != 750*time.Millisecond {
		t.Fatalf("task timeout = %s, err = %v", timeout, err)
	}
	timeout, err = benchmarkTimeout(3000)
	if err != nil || timeout != 2*time.Second {
		t.Fatalf("capped timeout = %s, err = %v", timeout, err)
	}
}

func TestBenchmarkDatasetRequiresSupportedColumnTypes(t *testing.T) {
	_, err := validatedBenchmarkDatasets(BenchmarkConfig{Dataset: &BenchmarkDataset{
		Table: "items",
		Columns: []BenchmarkColumn{
			{Name: "id", Type: "NUMERIC", Expression: "row_number"},
		},
	}})
	if err == nil {
		t.Fatal("validatedBenchmarkDatasets accepted unsupported NUMERIC type")
	}

	schema := benchmarkSchemaSQL(BenchmarkConfig{}, []BenchmarkDataset{{
		Table: "items",
		Columns: []BenchmarkColumn{
			{Name: "id", Type: "INTEGER", Expression: "row_number"},
			{Name: "label", Type: "TEXT", Expression: "'item'"},
		},
	}})
	if !strings.Contains(schema, `"id" INTEGER`) || !strings.Contains(schema, `"label" TEXT`) {
		t.Fatalf("generated schema = %q, want explicit column types", schema)
	}
}

func TestBenchmarkDatasetGeneratesValuesWithDeclaredStorageTypes(t *testing.T) {
	dataset := BenchmarkDataset{
		Table: "typed_values",
		Columns: []BenchmarkColumn{
			{Name: "integer_value", Type: "INTEGER", Expression: "'42'"},
			{Name: "real_value", Type: "REAL", Expression: "'1.5'"},
			{Name: "text_value", Type: "TEXT", Expression: "42"},
			{Name: "blob_value", Type: "BLOB", Expression: "'blob'"},
		},
	}
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	if _, err := db.Exec(benchmarkSchemaSQL(BenchmarkConfig{}, []BenchmarkDataset{dataset})); err != nil {
		t.Fatalf("create generated schema: %v", err)
	}
	insertSQL, err := benchmarkDataSQL(dataset, 1)
	if err != nil {
		t.Fatalf("benchmarkDataSQL returned error: %v", err)
	}
	if _, err := db.Exec(insertSQL); err != nil {
		t.Fatalf("insert generated values: %v", err)
	}

	var integerType, realType, textType, blobType string
	err = db.QueryRow(`SELECT typeof(integer_value), typeof(real_value), typeof(text_value), typeof(blob_value) FROM typed_values`).Scan(
		&integerType, &realType, &textType, &blobType,
	)
	if err != nil {
		t.Fatalf("read storage types: %v", err)
	}
	if integerType != "integer" || realType != "real" || textType != "text" || blobType != "blob" {
		t.Fatalf("storage types = %q, %q, %q, %q", integerType, realType, textType, blobType)
	}
}
