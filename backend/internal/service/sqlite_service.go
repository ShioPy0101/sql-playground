package service

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteService struct{}

func NewSQLiteService() *SQLiteService {
	return &SQLiteService{}
}

func (s *SQLiteService) Execute(csvText string, query string) (string, error) {
	// クエリ実行ごとに独立した SQLite DB を作る。
	tmpFile, err := os.CreateTemp("", "sqlite-playground-*.sqlite")
	if err != nil {
		return "", err
	}

	dbPath := tmpFile.Name()
	tmpFile.Close()

	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	// 単一 CSV は input、複数 CSV はセクション見出しのテーブル名で読み込む。
	if err := createInputTables(db, csvText); err != nil {
		return "", err
	}

	// 複数クエリを順に実行し、追跡できるように結果へ見出しを付ける。
	statements := splitSQLStatements(query)
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

func compactSQL(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}

type csvTable struct {
	name string
	csv  string
}

func createInputTables(db *sql.DB, csvText string) error {
	tables, err := parseCSVTables(csvText)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		normalizedName := strings.ToLower(table.name)
		if _, ok := seen[normalizedName]; ok {
			return fmt.Errorf("duplicate table name: %s", table.name)
		}
		seen[normalizedName] = struct{}{}

		if err := createCSVTable(db, table.name, table.csv); err != nil {
			return fmt.Errorf("table %s: %w", table.name, err)
		}
	}

	return nil
}

func parseCSVTables(csvText string) ([]csvTable, error) {
	var tables []csvTable
	currentName := "input"
	var currentCSV strings.Builder
	sawMarker := false

	for _, line := range strings.Split(csvText, "\n") {
		if tableName, ok := parseTableMarker(line); ok {
			csvPart := trimLineBreaks(currentCSV.String())
			if strings.TrimSpace(csvPart) != "" {
				if !sawMarker {
					return nil, fmt.Errorf("csv before first table marker is not supported")
				}
				tables = append(tables, csvTable{name: currentName, csv: csvPart})
			}

			currentName = tableName
			currentCSV.Reset()
			sawMarker = true
			continue
		}

		currentCSV.WriteString(line)
		currentCSV.WriteByte('\n')
	}

	csvPart := trimLineBreaks(currentCSV.String())
	if strings.TrimSpace(csvPart) != "" {
		tables = append(tables, csvTable{name: currentName, csv: csvPart})
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("csv is empty")
	}

	return tables, nil
}

func trimLineBreaks(text string) string {
	return strings.Trim(text, "\r\n")
}

func parseTableMarker(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
		if name != "" {
			return name, true
		}
	}

	for _, prefix := range []string{"# table:", "-- table:"} {
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			name := strings.TrimSpace(trimmed[len(prefix):])
			if name != "" {
				return name, true
			}
		}
	}

	return "", false
}

func createCSVTable(db *sql.DB, tableName string, csvText string) error {
	// 先頭行をヘッダとして扱い、すべて TEXT カラムで作成する。
	reader := csv.NewReader(strings.NewReader(csvText))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("csv is empty")
		}
		return err
	}
	if len(headers) == 0 {
		return fmt.Errorf("csv header is empty")
	}

	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			header = fmt.Sprintf("column_%d", i+1)
		}
		headers[i] = header
	}

	columnDefs := make([]string, len(headers))
	for i, header := range headers {
		columnDefs[i] = fmt.Sprintf("%s TEXT", quoteIdentifier(header))
	}

	if _, err := db.Exec(fmt.Sprintf("CREATE TABLE %s (%s)", quoteIdentifier(tableName), strings.Join(columnDefs, ", "))); err != nil {
		return err
	}

	placeholders := make([]string, len(headers))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(tableName),
		quoteIdentifiers(headers),
		strings.Join(placeholders, ", "),
	)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		values := make([]any, len(headers))
		for i := range headers {
			if i < len(record) {
				values[i] = record[i]
			} else {
				values[i] = ""
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func executeStatement(db *sql.DB, statement string) (string, error) {
	// DDL/DML は結果行を持たないため Exec で実行し、成功だけを返す。
	if !returnsRows(statement) {
		if _, err := db.Exec(statement); err != nil {
			return "", err
		}
		return "OK\n", nil
	}

	// SELECT などの結果行を持つ文は CSV 文字列へ変換する。
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

	var output bytes.Buffer
	writer := csv.NewWriter(&output)

	if err := writer.Write(columns); err != nil {
		return "", err
	}

	for rows.Next() {
		rawValues := make([]sql.NullString, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range rawValues {
			scanArgs[i] = &rawValues[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return "", err
		}

		record := make([]string, len(columns))
		for i, value := range rawValues {
			if value.Valid {
				record[i] = value.String
			}
		}

		if err := writer.Write(record); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return output.String(), nil
}

func returnsRows(statement string) bool {
	// 先頭キーワードで、結果セットを返すSQLかどうかを判定する。
	firstWord := strings.ToUpper(firstSQLWord(statement))
	switch firstWord {
	case "SELECT", "WITH", "VALUES", "PRAGMA", "EXPLAIN":
		return true
	default:
		return false
	}
}

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

func splitSQLStatements(query string) []string {
	// 文字列リテラルやコメント内のセミコロンでは分割しない。
	var statements []string
	var current strings.Builder
	var quote rune
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i, r := range query {
		next := rune(0)
		if i+1 < len(query) {
			next = rune(query[i+1])
		}

		if inLineComment {
			current.WriteRune(r)
			if r == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			current.WriteRune(r)
			if r == '*' && next == '/' {
				continue
			}
			if r == '/' && i > 0 && query[i-1] == '*' {
				inBlockComment = false
			}
			continue
		}

		if quote == 0 && r == '-' && next == '-' {
			inLineComment = true
			current.WriteRune(r)
			continue
		}
		if quote == 0 && r == '/' && next == '*' {
			inBlockComment = true
			current.WriteRune(r)
			continue
		}

		if quote != 0 {
			current.WriteRune(r)
			if r == quote && !escaped {
				quote = 0
			}
			escaped = r == '\\' && !escaped
			if r != '\\' {
				escaped = false
			}
			continue
		}

		if r == '\'' || r == '"' || r == '`' {
			quote = r
			current.WriteRune(r)
			continue
		}

		if r == ';' {
			statement := strings.TrimSpace(current.String())
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	statement := strings.TrimSpace(current.String())
	if statement != "" {
		statements = append(statements, statement)
	}

	return statements
}

func quoteIdentifiers(identifiers []string) string {
	quoted := make([]string, len(identifiers))
	for i, identifier := range identifiers {
		quoted[i] = quoteIdentifier(identifier)
	}
	return strings.Join(quoted, ", ")
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
