package service

import "github.com/ShioPy0101/sql-playground/pkg/service/helper"

// SQLiteService executes SQL against CSV-backed temporary SQLite tables.
type SQLiteService struct{}

// NewSQLiteService creates a SQLiteService instance.
func NewSQLiteService() *SQLiteService {
	return &SQLiteService{}
}

// Execute loads csvText into SQLite tables and returns the query result as CSV text.
func (s *SQLiteService) Execute(csvText string, query string) (string, error) {
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		return "", err
	}
	defer cleanup()
	defer db.Close()

	if err := helper.CreateInputTables(db, csvText); err != nil {
		return "", err
	}

	return helper.ExecuteStatements(db, helper.SplitSQLStatements(query))
}

// ExecuteAndInspect runs SQL, then returns the result of an inspection query on the same database.
func (s *SQLiteService) ExecuteAndInspect(csvText string, query string, inspectionQuery string) (string, error) {
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		return "", err
	}
	defer cleanup()
	defer db.Close()

	if err := helper.CreateInputTables(db, csvText); err != nil {
		return "", err
	}

	if _, err := helper.ExecuteStatements(db, helper.SplitSQLStatements(query)); err != nil {
		return "", err
	}

	return helper.ExecuteStatements(db, helper.SplitSQLStatements(inspectionQuery))
}
