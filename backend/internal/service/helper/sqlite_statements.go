package helper

import (
	"database/sql"
	"fmt"
	"strings"
)

// ExecuteStatements runs split SQL statements and annotates multi-query results.
func ExecuteStatements(db *sql.DB, statements []string) (string, error) {
	if len(statements) == 0 {
		return "", nil
	}

	results := make([]string, 0, len(statements))
	for i, stmt := range statements {
		result, err := executeStatement(db, stmt)
		if err != nil {
			return "", fmt.Errorf("query %d failed: %w", i+1, err)
		}

		if len(statements) > 1 {
			results = append(results, fmt.Sprintf("-- Query %d: %s\n%s", i+1, compactSQL(stmt), result))
			continue
		}
		results = append(results, result)
	}

	return strings.Join(results, "\n"), nil
}

// executeStatement chooses Query or Exec based on whether the statement returns rows.
func executeStatement(db *sql.DB, statement string) (string, error) {
	if !returnsRows(statement) {
		return executeCommand(db, statement)
	}

	rows, err := db.Query(statement)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "OK\n", nil
	}

	return rowsToCSV(rows, columns)
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
