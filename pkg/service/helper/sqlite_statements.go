package helper

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StatementMetrics contains measurements for one SQL statement.
// The SQLite counters remain nil when the driver cannot expose sqlite3_stmt_status.
type StatementMetrics struct {
	StatementIndex int      `json:"statementIndex"`
	Statement      string   `json:"statement"`
	DurationMS     float64  `json:"durationMs"`
	VMSteps        *int64   `json:"vmSteps"`
	FullScanSteps  *int64   `json:"fullScanSteps"`
	SortOperations *int64   `json:"sortOperations"`
	AutoIndexRows  *int64   `json:"autoIndexRows"`
	QueryPlan      []string `json:"queryPlan"`
}

// ExecutionMetrics contains totals and per-statement measurements.
type ExecutionMetrics struct {
	DurationMS     float64            `json:"durationMs"`
	VMSteps        *int64             `json:"vmSteps"`
	FullScanSteps  *int64             `json:"fullScanSteps"`
	SortOperations *int64             `json:"sortOperations"`
	AutoIndexRows  *int64             `json:"autoIndexRows"`
	Statements     []StatementMetrics `json:"statements"`
}

// StatementExecutionResult is the common result of executing SQL statements.
type StatementExecutionResult struct {
	CSV     string           `json:"csv"`
	Metrics ExecutionMetrics `json:"metrics"`
}

// ExecuteStatements runs split SQL statements and annotates multi-query results.
func ExecuteStatements(db *sql.DB, statements []string) (string, error) {
	result, err := ExecuteStatementsWithStats(db, statements)
	return result.CSV, err
}

// ExecuteStatementsWithStats executes statements and measures only their
// Query/Exec work. Database creation, input loading and query-plan inspection
// are deliberately outside the timer.
func ExecuteStatementsWithStats(db *sql.DB, statements []string) (StatementExecutionResult, error) {
	if len(statements) == 0 {
		return StatementExecutionResult{Metrics: ExecutionMetrics{Statements: []StatementMetrics{}}}, nil
	}

	results := make([]string, 0, len(statements))
	metrics := ExecutionMetrics{Statements: make([]StatementMetrics, 0, len(statements))}
	for i, stmt := range statements {
		returnsResult := statementReturnsRows(db, stmt)
		statementMetrics := StatementMetrics{
			StatementIndex: i + 1,
			Statement:      compactSQL(stmt),
			QueryPlan:      queryPlan(db, stmt, returnsResult),
		}
		// Keep the keyword check only as a fallback when EXPLAIN cannot classify
		// a statement that the existing executor already knows returns rows.
		result, duration, err := executeStatementMeasured(db, stmt, returnsResult || returnsRows(stmt))
		statementMetrics.DurationMS = float64(duration.Nanoseconds()) / float64(time.Millisecond)
		if err != nil {
			return StatementExecutionResult{}, fmt.Errorf("query %d failed: %w", i+1, err)
		}
		metrics.DurationMS += statementMetrics.DurationMS
		metrics.Statements = append(metrics.Statements, statementMetrics)

		if len(statements) > 1 {
			results = append(results, fmt.Sprintf("-- Query %d: %s\n%s", i+1, compactSQL(stmt), result))
			continue
		}
		results = append(results, result)
	}

	return StatementExecutionResult{
		CSV:     strings.Join(results, "\n"),
		Metrics: metrics,
	}, nil
}

// queryPlan asks SQLite whether the bytecode returns rows before requesting a
// query plan. Plan inspection is supplementary and never blocks execution.
func queryPlan(db *sql.DB, statement string, returnsResult bool) []string {
	if !returnsResult {
		return []string{}
	}

	rows, err := db.Query("EXPLAIN QUERY PLAN " + statement)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	plan := make([]string, 0)
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			return []string{}
		}
		plan = append(plan, detail)
	}
	if rows.Err() != nil {
		return []string{}
	}
	return plan
}

func statementReturnsRows(db *sql.DB, statement string) bool {
	rows, err := db.Query("EXPLAIN " + statement)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var address, p1, p2, p3, p5 int
		var opcode string
		var p4, comment sql.NullString
		if err := rows.Scan(&address, &opcode, &p1, &p2, &p3, &p4, &p5, &comment); err != nil {
			return false
		}
		if opcode == "ResultRow" {
			return true
		}
	}
	return false
}

func executeStatementMeasured(db *sql.DB, statement string, returnsResult bool) (string, time.Duration, error) {
	if !returnsResult {
		startedAt := time.Now()
		result, err := executeCommand(db, statement)
		return result, time.Since(startedAt), err
	}

	startedAt := time.Now()
	rows, err := db.Query(statement)
	if err != nil {
		return "", time.Since(startedAt), err
	}

	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return "", time.Since(startedAt), err
	}
	if len(columns) == 0 {
		err := rows.Close()
		return "OK\n", time.Since(startedAt), err
	}

	records, err := readRows(rows, len(columns))
	closeErr := rows.Close()
	duration := time.Since(startedAt)
	if err != nil {
		return "", duration, err
	}
	if closeErr != nil {
		return "", duration, closeErr
	}
	result, err := recordsToCSV(columns, records)
	return result, duration, err
}

func executeCommand(db *sql.DB, statement string) (string, error) {
	if _, err := db.Exec(statement); err != nil {
		return "", err
	}
	return "OK\n", nil
}

// compactSQL keeps multi-query trace headers on one display-friendly line.
func compactSQL(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}

// returnsRows checks the leading SQL keyword to decide how SQLite should execute it.
func returnsRows(statement string) bool {
	firstWord := strings.ToUpper(firstSQLWord(statement))
	switch firstWord {
	case "SELECT", "WITH", "VALUES", "PRAGMA", "EXPLAIN":
		return true
	default:
		return false
	}
}

// firstSQLWord skips leading SQL comments before reading the first keyword.
func firstSQLWord(statement string) string {
	statement = strings.TrimSpace(statement)
	for {
		switch {
		case strings.HasPrefix(statement, "--"):
			lineEnd := strings.IndexByte(statement, '\n')
			if lineEnd == -1 {
				return ""
			}
			statement = strings.TrimSpace(statement[lineEnd+1:])
		case strings.HasPrefix(statement, "/*"):
			commentEnd := strings.Index(statement, "*/")
			if commentEnd == -1 {
				return ""
			}
			statement = strings.TrimSpace(statement[commentEnd+2:])
		default:
			for i, r := range statement {
				if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' {
					return statement[:i]
				}
			}
			return statement
		}
	}
}
