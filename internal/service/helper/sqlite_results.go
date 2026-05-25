package helper

import (
	"bytes"
	"database/sql"
	"encoding/csv"
)

// rowsToCSV serializes a SELECT result set as CSV text.
func rowsToCSV(rows *sql.Rows, columns []string) (string, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)

	if err := writer.Write(columns); err != nil {
		return "", err
	}

	for rows.Next() {
		record, err := scanCSVRecord(rows, len(columns))
		if err != nil {
			return "", err
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
