package service

import "github.com/ShioPy0101/sql-playground/pkg/service/helper"

// SQLiteService executes SQL against CSV-backed temporary SQLite tables.
type SQLiteService struct{}

type ExecutionMetrics = helper.ExecutionMetrics
type StatementExecutionResult = helper.StatementExecutionResult

// NewSQLiteService creates a SQLiteService instance.
func NewSQLiteService() *SQLiteService {
	return &SQLiteService{}
}

// Execute loads CSV into SQLite tables and returns the query result as CSV text.
func (s *SQLiteService) Execute(csvText string, query string) (string, error) {
	return s.ExecuteInput(csvText, helper.InputFormatCSV, query)
}

// ExecuteInput parses input into a temporary SQLite database, then executes the query.
func (s *SQLiteService) ExecuteInput(input string, inputFormat string, query string) (string, error) {
	result, err := s.ExecuteInputWithStats(input, inputFormat, query)
	return result.CSV, err
}

// ExecuteInputWithStats returns the query result and statement-level execution statistics.
func (s *SQLiteService) ExecuteInputWithStats(input string, inputFormat string, query string) (helper.StatementExecutionResult, error) {
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		return helper.StatementExecutionResult{}, err
	}
	defer cleanup()
	defer db.Close()

	if err := helper.CreateInputDatabase(db, input, inputFormat); err != nil {
		return helper.StatementExecutionResult{}, err
	}

	return helper.ExecuteStatementsWithStats(db, helper.SplitSQLStatements(query))
}

// ExecuteAndInspect runs SQL, then returns the result of an inspection query on the same database.
func (s *SQLiteService) ExecuteAndInspect(csvText string, query string, inspectionQuery string) (string, error) {
	return s.ExecuteInputAndInspect(csvText, helper.InputFormatCSV, query, inspectionQuery)
}

// ExecuteInputAndInspect initializes either input format before query and inspection.
func (s *SQLiteService) ExecuteInputAndInspect(input string, inputFormat string, query string, inspectionQuery string) (string, error) {
	result, err := s.ExecuteInputAndInspectWithStats(input, inputFormat, query, inspectionQuery)
	return result.CSV, err
}

// ExecuteInputAndInspectWithStats measures the submitted query and returns the
// inspection query's CSV result, preserving the existing grading behavior.
func (s *SQLiteService) ExecuteInputAndInspectWithStats(input string, inputFormat string, query string, inspectionQuery string) (helper.StatementExecutionResult, error) {
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		return helper.StatementExecutionResult{}, err
	}
	defer cleanup()
	defer db.Close()

	if err := helper.CreateInputDatabase(db, input, inputFormat); err != nil {
		return helper.StatementExecutionResult{}, err
	}

	execution, err := helper.ExecuteStatementsWithStats(db, helper.SplitSQLStatements(query))
	if err != nil {
		return helper.StatementExecutionResult{}, err
	}

	inspectionCSV, err := helper.ExecuteStatements(db, helper.SplitSQLStatements(inspectionQuery))
	if err != nil {
		return helper.StatementExecutionResult{}, err
	}
	execution.CSV = inspectionCSV
	return execution, nil
}
