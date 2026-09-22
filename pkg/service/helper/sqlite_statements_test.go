package helper

import (
	"math"
	"strings"
	"testing"
)

func TestExecuteStatementsWithStatsReturnsPerStatementMeasurementsAndPlan(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO users VALUES (1, 'Ada'), (2, 'Linus')`); err != nil {
		t.Fatalf("initialize database: %v", err)
	}

	result, err := ExecuteStatementsWithStats(db, []string{
		`CREATE INDEX idx_users_name ON users(name)`,
		`SELECT name FROM users WHERE name = 'Ada'`,
	})
	if err != nil {
		t.Fatalf("ExecuteStatementsWithStats returned error: %v", err)
	}

	if len(result.Metrics.Statements) != 2 {
		t.Fatalf("statement metrics length = %d, want 2", len(result.Metrics.Statements))
	}
	if result.Metrics.DurationMS < 0 {
		t.Fatalf("duration = %f, want non-negative", result.Metrics.DurationMS)
	}
	wantTotal := result.Metrics.Statements[0].DurationMS + result.Metrics.Statements[1].DurationMS
	if math.Abs(result.Metrics.DurationMS-wantTotal) > 0.000001 {
		t.Fatalf("total duration = %f, want statement sum %f", result.Metrics.DurationMS, wantTotal)
	}
	if len(result.Metrics.Statements[0].QueryPlan) != 0 {
		t.Fatalf("CREATE INDEX plan = %#v, want empty", result.Metrics.Statements[0].QueryPlan)
	}
	plan := strings.Join(result.Metrics.Statements[1].QueryPlan, "\n")
	if !strings.Contains(plan, "idx_users_name") {
		t.Fatalf("SELECT plan = %q, want index name", plan)
	}
	if result.Metrics.VMSteps != nil || result.Metrics.FullScanSteps != nil ||
		result.Metrics.SortOperations != nil || result.Metrics.AutoIndexRows != nil {
		t.Fatal("sqlite3_stmt_status counters must be nil when unavailable")
	}
}

func TestExecuteStatementsWithStatsMeasuresDDLAndInsertWithoutPlans(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	result, err := ExecuteStatementsWithStats(db, []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`,
		`INSERT INTO users (id, name) VALUES (1, 'Ada')`,
	})
	if err != nil {
		t.Fatalf("ExecuteStatementsWithStats returned error: %v", err)
	}
	if len(result.Metrics.Statements) != 2 {
		t.Fatalf("statement metrics length = %d, want 2", len(result.Metrics.Statements))
	}
	for _, statement := range result.Metrics.Statements {
		if statement.DurationMS < 0 {
			t.Fatalf("statement %d duration = %f, want non-negative", statement.StatementIndex, statement.DurationMS)
		}
		if len(statement.QueryPlan) != 0 {
			t.Fatalf("statement %d plan = %#v, want empty", statement.StatementIndex, statement.QueryPlan)
		}
	}
	if !strings.Contains(result.CSV, "-- Query 1: CREATE TABLE") ||
		!strings.Contains(result.CSV, "-- Query 2: INSERT INTO") {
		t.Fatalf("CSV trace changed unexpectedly: %q", result.CSV)
	}
}

func TestExecuteStatementsWithStatsPlansNonSelectResultStatement(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	result, err := ExecuteStatementsWithStats(db, []string{`VALUES (1), (2)`})
	if err != nil {
		t.Fatalf("ExecuteStatementsWithStats returned error: %v", err)
	}
	if len(result.Metrics.Statements[0].QueryPlan) == 0 {
		t.Fatal("VALUES should receive a query plan based on SQLite bytecode")
	}
}

func TestExecuteStatementsWithStatsIgnoresUnsupportedQueryPlan(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	result, err := ExecuteStatementsWithStats(db, []string{`PRAGMA table_info(users)`})
	if err != nil {
		t.Fatalf("statement must succeed when EXPLAIN QUERY PLAN is unsupported: %v", err)
	}
	if !strings.Contains(result.CSV, "name,type") {
		t.Fatalf("CSV = %q, want PRAGMA result", result.CSV)
	}
	if len(result.Metrics.Statements[0].QueryPlan) != 0 {
		t.Fatalf("query plan = %#v, want empty", result.Metrics.Statements[0].QueryPlan)
	}
}

func TestExecuteStatementsWithStatsDoesNotIncludeQueryPlanAsStatement(t *testing.T) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	result, err := ExecuteStatementsWithStats(db, []string{`SELECT 1 AS value`})
	if err != nil {
		t.Fatalf("ExecuteStatementsWithStats returned error: %v", err)
	}
	if len(result.Metrics.Statements) != 1 {
		t.Fatalf("statement metrics length = %d, want 1", len(result.Metrics.Statements))
	}
	if result.CSV != "value\n1\n" {
		t.Fatalf("CSV = %q, want query result only", result.CSV)
	}
}
