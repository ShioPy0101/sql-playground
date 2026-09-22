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

var benchmarkSubmissionSelectPattern = regexp.MustCompile(`(?is)^\s*(?:(?:--[^\n]*(?:\n|$))|(?:/\*.*?\*/\s*))*(?:SELECT|WITH)\b`)
var benchmarkFixedSelectPattern = regexp.MustCompile(`(?is)^\s*(?:(?:--[^\n]*(?:\n|$))|(?:/\*.*?\*/\s*))*SELECT\b`)

var benchmarkDDLPattern = regexp.MustCompile(`(?is)^\s*(?:(?:--[^\n]*(?:\n|$))|(?:/\*.*?\*/\s*))*CREATE\s+(?:TABLE|(?:UNIQUE\s+)?INDEX)\b`)
var benchmarkCommentsOnlyPattern = regexp.MustCompile(`(?is)^\s*(?:(?:--[^\n]*(?:\n|$))|(?:/\*.*?\*/\s*))*$`)
var benchmarkIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var benchmarkColumnTypes = map[string]bool{"INTEGER": true, "REAL": true, "TEXT": true, "BLOB": true}

var DefaultBenchmarkRowCounts = []int{1_000, 10_000, 50_000, 100_000}

type BenchmarkConfig struct {
	Enabled     bool               `json:"enabled"`
	RunOnFailed bool               `json:"runOnFailed,omitempty"`
	TimeoutMS   int                `json:"timeoutMs,omitempty"`
	Target      string             `json:"target"`
	Query       string             `json:"query,omitempty"`
	RowCounts   []int              `json:"rowCounts"`
	SchemaSQL   string             `json:"schemaSql,omitempty"`
	Dataset     *BenchmarkDataset  `json:"dataset,omitempty"`
	Datasets    []BenchmarkDataset `json:"datasets,omitempty"`
}

type BenchmarkDataset struct {
	Table   string            `json:"table"`
	Columns []BenchmarkColumn `json:"columns"`
}

type BenchmarkColumn struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
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
	datasets, err := validatedBenchmarkDatasets(config)
	if err != nil {
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
		result, err := runBenchmarkStage(ctx, rowCount, taskMode, setupSQL, query, config, datasets)
		if err != nil {
			report.Status = "failed"
			report.StoppedReason = fmt.Sprintf("%d行の計測で停止しました: %v", rowCount, err)
			break
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func runBenchmarkStage(ctx context.Context, rowCount int, taskMode string, setupSQL string, query string, config BenchmarkConfig, datasets []BenchmarkDataset) (BenchmarkResult, error) {
	db, databasePath, cleanup, err := helper.OpenTempSQLiteDBWithPath()
	if err != nil {
		return BenchmarkResult{}, err
	}
	defer cleanup()
	defer db.Close()
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return BenchmarkResult{}, fmt.Errorf("外部キー有効化: %w", err)
	}

	if taskMode == "ddl" {
		if _, err := db.ExecContext(ctx, setupSQL); err != nil {
			return BenchmarkResult{}, fmt.Errorf("受講者DDL適用: %w", err)
		}
	} else if _, err := db.ExecContext(ctx, benchmarkSchemaSQL(config, datasets)); err != nil {
		return BenchmarkResult{}, fmt.Errorf("計測用schema作成: %w", err)
	}

	generationStartedAt := time.Now()
	for _, dataset := range datasets {
		dataSQL, err := benchmarkDataSQL(dataset, rowCount)
		if err != nil {
			return BenchmarkResult{}, err
		}
		if _, err := db.ExecContext(ctx, dataSQL); err != nil {
			return BenchmarkResult{}, fmt.Errorf("%sのテストデータ生成: %w", dataset.Table, err)
		}
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
		columnType := strings.ToUpper(strings.TrimSpace(column.Type))
		if !benchmarkIdentifierPattern.MatchString(column.Name) || !benchmarkColumnTypes[columnType] || strings.TrimSpace(column.Expression) == "" {
			return "", fmt.Errorf("benchmark.datasetの列定義が不正です")
		}
		columnNames = append(columnNames, quoteBenchmarkIdentifier(column.Name))
		expressions = append(expressions, fmt.Sprintf("CAST((%s) AS %s)", column.Expression, columnType))
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

func benchmarkSchemaSQL(config BenchmarkConfig, datasets []BenchmarkDataset) string {
	if strings.TrimSpace(config.SchemaSQL) != "" {
		return config.SchemaSQL
	}
	statements := make([]string, 0, len(datasets))
	for _, dataset := range datasets {
		columns := make([]string, 0, len(dataset.Columns))
		for _, column := range dataset.Columns {
			columns = append(columns, fmt.Sprintf("%s %s", quoteBenchmarkIdentifier(column.Name), strings.ToUpper(strings.TrimSpace(column.Type))))
		}
		statements = append(statements, fmt.Sprintf("CREATE TABLE %s (%s);", quoteBenchmarkIdentifier(dataset.Table), strings.Join(columns, ", ")))
	}
	return strings.Join(statements, "\n")
}

func benchmarkInputs(taskMode string, config BenchmarkConfig, submission string) (query string, setupSQL string, err error) {
	switch {
	case taskMode == "sql" && config.Target == "submission":
		statements := helper.SplitSQLStatements(submission)
		if len(statements) != 1 || !matchesBenchmarkSubmissionSelect(statements[0]) {
			return "", "", fmt.Errorf("benchmark対象は安全な単一SELECTに限定されています")
		}
		return strings.TrimSpace(statements[0]), "", nil
	case (taskMode == "ddl" || taskMode == "sql") && config.Target == "fixed-query":
		if !matchesBenchmarkFixedSelect(config.Query) {
			return "", "", fmt.Errorf("benchmark.queryは許可されたSELECTではありません")
		}
		if taskMode == "sql" {
			return strings.TrimSpace(config.Query), "", nil
		}
		statements := helper.SplitSQLStatements(submission)
		if len(statements) == 0 {
			return "", "", fmt.Errorf("受講者DDLが空です")
		}
		ddlCount := 0
		for _, statement := range statements {
			if benchmarkCommentsOnlyPattern.MatchString(statement) {
				continue
			}
			if !benchmarkDDLPattern.MatchString(statement) {
				return "", "", fmt.Errorf("DDL benchmarkではCREATE TABLE / CREATE INDEXのみ利用できます")
			}
			ddlCount++
		}
		if ddlCount == 0 {
			return "", "", fmt.Errorf("受講者DDLが空です")
		}
		return strings.TrimSpace(config.Query), submission, nil
	default:
		return "", "", fmt.Errorf("benchmarkのmodeとtargetの組み合わせが不正です")
	}
}

func matchesBenchmarkSubmissionSelect(statement string) bool {
	statements := helper.SplitSQLStatements(statement)
	return len(statements) == 1 && benchmarkSubmissionSelectPattern.MatchString(statements[0])
}

func matchesBenchmarkFixedSelect(statement string) bool {
	statements := helper.SplitSQLStatements(statement)
	return len(statements) == 1 && benchmarkFixedSelectPattern.MatchString(statements[0])
}

func validatedBenchmarkDatasets(config BenchmarkConfig) ([]BenchmarkDataset, error) {
	datasets := config.Datasets
	if len(datasets) == 0 && config.Dataset != nil {
		datasets = []BenchmarkDataset{*config.Dataset}
	}
	if len(datasets) == 0 {
		return nil, fmt.Errorf("benchmark.datasetまたはbenchmark.datasetsが設定されていません")
	}
	for _, dataset := range datasets {
		if !benchmarkIdentifierPattern.MatchString(dataset.Table) || len(dataset.Columns) == 0 {
			return nil, fmt.Errorf("benchmark.datasetの定義が不正です")
		}
		for _, column := range dataset.Columns {
			if !benchmarkIdentifierPattern.MatchString(column.Name) || !benchmarkColumnTypes[strings.ToUpper(strings.TrimSpace(column.Type))] || strings.TrimSpace(column.Expression) == "" {
				return nil, fmt.Errorf("benchmark.datasetの列定義が不正です")
			}
		}
	}
	return datasets, nil
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
