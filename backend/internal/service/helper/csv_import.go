package helper

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// createCSVTable imports one CSV section as a SQLite table with TEXT columns.
func createCSVTable(db *sql.DB, tableName string, csvText string) error {
	reader := newCSVReader(csvText)
	headers, err := readCSVHeaders(reader)
	if err != nil {
		return err
	}

	if err := createTable(db, tableName, headers); err != nil {
		return err
	}

	return insertCSVRecords(db, reader, tableName, headers)
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

// createTable creates a SQLite table using the CSV headers as TEXT columns.
func createTable(db *sql.DB, tableName string, headers []string) error {
	columnDefs := make([]string, len(headers))
	for i, header := range headers {
		columnDefs[i] = fmt.Sprintf("%s TEXT", quoteIdentifier(header))
	}

	_, err := db.Exec(fmt.Sprintf(
		"CREATE TABLE %s (%s)",
		quoteIdentifier(tableName),
		strings.Join(columnDefs, ", "),
	))
	return err
}

// insertCSVRecords inserts remaining CSV rows, padding short rows with empty strings.
func insertCSVRecords(db *sql.DB, reader *csv.Reader, tableName string, headers []string) error {
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

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if _, err := stmt.Exec(csvRecordValues(record, len(headers))...); err != nil {
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

func csvRecordValues(record []string, columnCount int) []any {
	values := make([]any, columnCount)
	for i := range columnCount {
		if i < len(record) {
			values[i] = record[i]
		} else {
			values[i] = ""
		}
	}
	return values
}
