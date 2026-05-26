package helper

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// createCSVTable imports one CSV section as a SQLite table.
func createCSVTable(db *sql.DB, tableName string, csvText string) error {
	reader := newCSVReader(csvText)
	headers, err := readCSVHeaders(reader)
	if err != nil {
		return err
	}
	records, err := readCSVRecords(reader)
	if err != nil {
		return err
	}

	columnTypes := inferColumnTypes(records, len(headers))
	if err := createTable(db, tableName, headers, columnTypes); err != nil {
		return err
	}

	return insertCSVRecords(db, records, tableName, headers, columnTypes)
}

func newCSVReader(csvText string) *csv.Reader {
	reader := csv.NewReader(strings.NewReader(csvText))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	return reader
}

// readCSVHeaders reads the first CSV row and normalizes empty column names.
func readCSVHeaders(reader *csv.Reader) ([]string, error) {
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("csv is empty")
		}
		return nil, err
	}
	if len(headers) == 0 {
		return nil, fmt.Errorf("csv header is empty")
	}

	return normalizeHeaders(headers), nil
}

func normalizeHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			header = fmt.Sprintf("column_%d", i+1)
		}
		normalized[i] = header
	}
	return normalized
}

// readCSVRecords reads the CSV body after the header row.
func readCSVRecords(reader *csv.Reader) ([][]string, error) {
	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func inferColumnTypes(records [][]string, columnCount int) []string {
	columnTypes := make([]string, columnCount)
	for column := range columnTypes {
		columnTypes[column] = "TEXT"
		if columnHasOnlyIntegers(records, column) {
			columnTypes[column] = "INTEGER"
		}
	}
	return columnTypes
}

func columnHasOnlyIntegers(records [][]string, column int) bool {
	if len(records) == 0 {
		return false
	}

	for _, record := range records {
		if column >= len(record) || !isIntegerLiteral(record[column]) {
			return false
		}
	}
	return true
}

func isIntegerLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return false
	}

	unsigned := strings.TrimPrefix(strings.TrimPrefix(value, "-"), "+")
	return len(unsigned) == 1 || !strings.HasPrefix(unsigned, "0")
}

// createTable creates a SQLite table using inferred CSV column types.
func createTable(db *sql.DB, tableName string, headers []string, columnTypes []string) error {
	columnDefs := make([]string, len(headers))
	for i, header := range headers {
		columnDefs[i] = fmt.Sprintf("%s %s", quoteIdentifier(header), columnTypes[i])
	}

	_, err := db.Exec(fmt.Sprintf(
		"CREATE TABLE %s (%s)",
		quoteIdentifier(tableName),
		strings.Join(columnDefs, ", "),
	))
	return err
}

// insertCSVRecords inserts CSV rows, padding short rows with empty strings.
func insertCSVRecords(db *sql.DB, records [][]string, tableName string, headers []string, columnTypes []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(insertSQL(tableName, headers))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, record := range records {
		if _, err := stmt.Exec(csvRecordValues(record, len(headers), columnTypes)...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func insertSQL(tableName string, headers []string) string {
	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(tableName),
		quoteIdentifiers(headers),
		placeholders(len(headers)),
	)
}

func placeholders(count int) string {
	values := make([]string, count)
	for i := range values {
		values[i] = "?"
	}
	return strings.Join(values, ", ")
}

func csvRecordValues(record []string, columnCount int, columnTypes []string) []any {
	values := make([]any, columnCount)
	for i := range columnCount {
		if i >= len(record) {
			values[i] = ""
			continue
		}

		value := record[i]
		if columnTypes[i] == "INTEGER" {
			if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
				values[i] = parsed
				continue
			}
		}
		values[i] = value
	}
	return values
}
