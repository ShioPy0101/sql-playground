package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ShioPy0101/sql-playground/pkg/service/helper"
)

var benchmarkSelectPattern = regexp.MustCompile(`(?is)^\s*SELECT\s+\*\s+FROM\s+([A-Za-z_][A-Za-z0-9_]*)\s+WHERE\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*-?[0-9]+(?:\s+AND\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*-?[0-9]+)*\s*$`)

var benchmarkDDLPattern = regexp.MustCompile(`(?is)^\s*(?:(?:--[^\n]*(?:\n|$))|(?:/\*.*?\*/\s*))*CREATE\s+(?:TABLE|(?:UNIQUE\s+)?INDEX)\b`)
var benchmarkIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var DefaultBenchmarkRowCounts = []int{1_000, 10_000, 50_000, 100_000}

type BenchmarkConfig struct {
	Enabled     bool              `json:"enabled"`
	RunOnFailed bool              `json:"runOnFailed,omitempty"`
	TimeoutMS   int               `json:"timeoutMs,omitempty"`
	Target      string            `json:"target"`
	Query       string            `json:"query,omitempty"`
	RowCounts   []int             `json:"rowCounts"`
	SchemaSQL   string            `json:"schemaSql,omitempty"`
	Dataset     *BenchmarkDataset `json:"dataset,omitempty"`
}

type BenchmarkDataset struct {
	Table   string            `json:"table"`
	Columns []BenchmarkColumn `json:"columns"`
}

type BenchmarkColumn struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type BenchmarkResult struct {
	RowCount          int      `json:"rowCount"`
	ExecutionTimeMS   float64  `json:"executionTimeMs"`
	VMSteps           int64    `json:"vmSteps"`
	FullScanSteps     int64    `json:"fullScanSteps"`
	SortCount         int64    `json:"sortCount"`
	AutoIndexRows     int64    `json:"autoIndexRows"`
	QueryPlan         []string `json:"queryPlan"`
	DatabaseSizeBytes int64    `json:"databaseSizeBytes"`
	DataGenerationMS  float64  `json:"dataGenerationMs"`
}

type BenchmarkReport struct {
	Status        string            `json:"status"`
	Query         string            `json:"query"`
	Results       []BenchmarkResult `json:"results"`
	StoppedReason string            `json:"error,omitempty"`
}

type BenchmarkService struct{}

func NewBenchmarkService() *BenchmarkService {
	return &BenchmarkService{}
}

// Run is an explicit benchmark operation. SQLiteService and TaskService never
// call it, so normal execution, grading and submission persistence cannot
// trigger large data generation.
func (s *BenchmarkService) Run(taskMode string, config BenchmarkConfig, submission string) (BenchmarkReport, error) {
	return s.RunContext(context.Background(), taskMode, config, submission)
}

func (s *BenchmarkService) RunContext(parent context.Context, taskMode string, config BenchmarkConfig, submission string) (BenchmarkReport, error) {
	query, setupSQL, err := benchmarkInputs(taskMode, config, submission)
	if err != nil {
		return BenchmarkReport{}, err
	}
	if taskMode == "sql" && strings.TrimSpace(config.SchemaSQL) == "" {
		return BenchmarkReport{}, fmt.Errorf("SQL問題のbenchmark.schemaSqlが設定されていません")
	}
	if err := validateBenchmarkDataset(config.Dataset, query); err != nil {
		return BenchmarkReport{}, err
	}
	rowCounts, err := validatedBenchmarkRowCounts(config.RowCounts)
	if err != nil {
		return BenchmarkReport{}, err
	}
	timeout, err := benchmarkTimeout(config.TimeoutMS)
	if err != nil {
		return BenchmarkReport{}, err
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	report := BenchmarkReport{Status: "completed", Query: query, Results: make([]BenchmarkResult, 0, len(rowCounts))}
	for _, rowCount := range rowCounts {
		result, err := runBenchmarkStage(ctx, rowCount, taskMode, setupSQL, query, config)
		if err != nil {
			report.Status = "failed"
			report.StoppedReason = fmt.Sprintf("%d行の計測で停止しました: %v", rowCount, err)
			break
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func runBenchmarkStage(ctx context.Context, rowCount int, taskMode string, setupSQL string, query string, config BenchmarkConfig) (BenchmarkResult, error) {
	db, databasePath, cleanup, err := helper.OpenTempSQLiteDBWithPath()
	if err != nil {
		return BenchmarkResult{}, err
	}
	defer cleanup()
	defer db.Close()

	if taskMode == "ddl" {
		if _, err := db.ExecContext(ctx, setupSQL); err != nil {
			return BenchmarkResult{}, fmt.Errorf("受講者DDL適用: %w", err)
		}
	} else if _, err := db.ExecContext(ctx, config.SchemaSQL); err != nil {
		return BenchmarkResult{}, fmt.Errorf("計測用schema作成: %w", err)
	}

	generationStartedAt := time.Now()
	dataSQL, err := benchmarkDataSQL(*config.Dataset, rowCount)
	if err != nil {
		return BenchmarkResult{}, err
	}
	if _, err := db.ExecContext(ctx, dataSQL); err != nil {
		return BenchmarkResult{}, fmt.Errorf("テストデータ生成: %w", err)
	}
	generationDuration := time.Since(generationStartedAt)
	if err := ctx.Err(); err != nil {
		return BenchmarkResult{}, fmt.Errorf("計測準備が制限時間を超えました: %w", err)
	}

	plan, err := benchmarkQueryPlan(ctx, db, query)
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("実行計画取得: %w", err)
	}
	status, err := helper.MeasureSelectStatementContext(ctx, databasePath, query)
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("SELECT計測: %w", err)
	}

	databaseSize := int64(0)
	if info, err := os.Stat(databasePath); err == nil {
		databaseSize = info.Size()
	}
	return BenchmarkResult{
		RowCount:          rowCount,
		ExecutionTimeMS:   status.DurationMS,
		VMSteps:           status.VMSteps,
		FullScanSteps:     status.FullScanSteps,
		SortCount:         status.SortOperations,
		AutoIndexRows:     status.AutoIndexRows,
		QueryPlan:         plan,
		DatabaseSizeBytes: databaseSize,
		DataGenerationMS:  float64(generationDuration.Nanoseconds()) / float64(time.Millisecond),
	}, nil
}

func benchmarkDataSQL(dataset BenchmarkDataset, rowCount int) (string, error) {
	columnNames := make([]string, 0, len(dataset.Columns))
	expressions := make([]string, 0, len(dataset.Columns))
	for _, column := range dataset.Columns {
		if !benchmarkIdentifierPattern.MatchString(column.Name) || strings.TrimSpace(column.Expression) == "" {
			return "", fmt.Errorf("benchmark.datasetの列定義が不正です")
		}
		columnNames = append(columnNames, quoteBenchmarkIdentifier(column.Name))
		expressions = append(expressions, column.Expression)
	}

	return fmt.Sprintf(`
		WITH RECURSIVE sequence(row_number) AS (
			SELECT 1
			UNION ALL
			SELECT row_number + 1 FROM sequence WHERE row_number < %d
		)
		INSERT INTO %s (%s)
		SELECT %s
		FROM sequence;
	`, rowCount, quoteBenchmarkIdentifier(dataset.Table), strings.Join(columnNames, ", "), strings.Join(expressions, ", ")), nil
}

func benchmarkInputs(taskMode string, config BenchmarkConfig, submission string) (query string, setupSQL string, err error) {
	switch {
	case taskMode == "sql" && config.Target == "submission":
		statements := helper.SplitSQLStatements(submission)
		if len(statements) != 1 || !matchesBenchmarkSelect(statements[0]) {
			return "", "", fmt.Errorf("benchmark対象は安全な単一SELECTに限定されています")
		}
		return strings.TrimSpace(statements[0]), "", nil
	case taskMode == "ddl" && config.Target == "fixed-query":
		if !matchesBenchmarkSelect(config.Query) {
			return "", "", fmt.Errorf("benchmark.queryは許可されたSELECTではありません")
		}
		statements := helper.SplitSQLStatements(submission)
		if len(statements) == 0 {
			return "", "", fmt.Errorf("受講者DDLが空です")
		}
		for _, statement := range statements {
			if !benchmarkDDLPattern.MatchString(statement) {
				return "", "", fmt.Errorf("DDL benchmarkではCREATE TABLE / CREATE INDEXのみ利用できます")
			}
		}
		return strings.TrimSpace(config.Query), submission, nil
	default:
		return "", "", fmt.Errorf("benchmarkのmodeとtargetの組み合わせが不正です")
	}
}

func benchmarkTable(query string) string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(query), ";"))
	matches := benchmarkSelectPattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func matchesBenchmarkSelect(statement string) bool {
	return benchmarkTable(statement) != ""
}

func validateBenchmarkDataset(dataset *BenchmarkDataset, query string) error {
	if dataset == nil || !benchmarkIdentifierPattern.MatchString(dataset.Table) || len(dataset.Columns) == 0 {
		return fmt.Errorf("benchmark.datasetが設定されていません")
	}
	if !strings.EqualFold(dataset.Table, benchmarkTable(query)) {
		return fmt.Errorf("benchmark.dataset.tableとqueryの対象テーブルが一致しません")
	}
	return nil
}

func quoteBenchmarkIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func validatedBenchmarkRowCounts(requested []int) ([]int, error) {
	if len(requested) == 0 {
		requested = DefaultBenchmarkRowCounts
	}
	maxRows := 100_000
	if configured := os.Getenv("SQLITE_BENCHMARK_MAX_ROWS"); configured != "" {
		value, err := strconv.Atoi(configured)
		if err != nil || value < 1 {
			return nil, fmt.Errorf("SQLITE_BENCHMARK_MAX_ROWSが不正です")
		}
		maxRows = value
	}

	result := make([]int, len(requested))
	previous := 0
	for index, rowCount := range requested {
		if rowCount <= previous || rowCount > maxRows {
			return nil, fmt.Errorf("rowCountsは昇順かつ最大%d行以下で指定してください", maxRows)
		}
		result[index] = rowCount
		previous = rowCount
	}
	return result, nil
}

func benchmarkTimeout(requestedMilliseconds int) (time.Duration, error) {
	if requestedMilliseconds < 0 {
		return 0, fmt.Errorf("benchmark.timeoutMsは正の整数で指定してください")
	}
	if requestedMilliseconds == 0 {
		requestedMilliseconds = 5_000
	}

	maximumMilliseconds := 5_000
	configured := os.Getenv("SQLITE_BENCHMARK_TIMEOUT_MS")
	if configured == "" {
		configured = os.Getenv("SQLITE_BENCHMARK_STAGE_TIMEOUT_MS")
	}
	if configured != "" {
		value, err := strconv.Atoi(configured)
		if err != nil || value < 1 {
			return 0, fmt.Errorf("SQLITE_BENCHMARK_TIMEOUT_MSが不正です")
		}
		maximumMilliseconds = value
	}
	if requestedMilliseconds > maximumMilliseconds {
		requestedMilliseconds = maximumMilliseconds
	}
	return time.Duration(requestedMilliseconds) * time.Millisecond, nil
}

func benchmarkQueryPlan(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plan := make([]string, 0)
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			return nil, err
		}
		plan = append(plan, detail)
	}
	return plan, rows.Err()
}
