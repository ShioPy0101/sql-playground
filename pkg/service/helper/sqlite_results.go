package helper

import (
	"bytes"
	"database/sql"
	"encoding/csv"
)

// readRows reads every SQLite row. Keep this separate from CSV serialization so
// statement timing can stop as soon as SQLite has produced the complete result.
func readRows(rows *sql.Rows, columnCount int) ([][]string, error) {
	records := make([][]string, 0)
	for rows.Next() {
		record, err := scanCSVRecord(rows, columnCount)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// recordsToCSV serializes values already read from SQLite.
func recordsToCSV(columns []string, records [][]string) (string, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)

	if err := writer.Write(columns); err != nil {
		return "", err
	}

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return output.String(), nil
}

// scanCSVRecord converts nullable SQLite values into CSV cells.
func scanCSVRecord(rows *sql.Rows, columnCount int) ([]string, error) {
	rawValues := make([]sql.NullString, columnCount)
	scanArgs := make([]any, columnCount)
	for i := range rawValues {
		scanArgs[i] = &rawValues[i]
	}

	if err := rows.Scan(scanArgs...); err != nil {
		return nil, err
	}

	record := make([]string, columnCount)
	for i, value := range rawValues {
		if value.Valid {
			record[i] = value.String
		}
	}

	return record, nil
}
