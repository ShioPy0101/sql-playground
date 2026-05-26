package helper

import (
	"database/sql"
	"fmt"
	"strings"
)

type csvTable struct {
	name string
	csv  string
}

// CreateInputTables supports a single default input table or multiple marked tables.
func CreateInputTables(db *sql.DB, csvText string) error {
	if strings.TrimSpace(csvText) == "" {
		return nil
	}

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

// parseCSVTables reads table sections marked by "# table:", "-- table:", or "[name]".
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

// parseTableMarker returns a table name when the line declares a CSV section.
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

func trimLineBreaks(text string) string {
	return strings.Trim(text, "\r\n")
}
