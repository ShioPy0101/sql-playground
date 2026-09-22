package helper

import (
	"database/sql"
	"fmt"
	"strings"
)

const (
	InputFormatCSV = "csv"
	InputFormatSQL = "sql"
)

// CreateInputDatabase parses either supported input format into the same
// temporary SQLite database used by the query executor.
func CreateInputDatabase(db *sql.DB, source string, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", InputFormatCSV:
		return CreateInputTables(db, source)
	case InputFormatSQL:
		return executeSQLInput(db, source)
	default:
		return fmt.Errorf("unsupported input format: %s", format)
	}
}

// executeSQLInput lets SQLite parse the input schema and data. Running all
// statements in one transaction prevents a partly initialized database.
func executeSQLInput(db *sql.DB, source string) error {
	statements := SplitSQLStatements(source)
	if len(statements) == 0 {
		return nil
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for index, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("input statement %d failed: %w", index+1, err)
		}
	}

	return tx.Commit()
}
