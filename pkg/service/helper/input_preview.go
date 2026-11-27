package helper

// InputTable contains the column names followed by the initial rows of a table.
type InputTable struct {
	Name string     `json:"name"`
	Rows [][]string `json:"rows"`
}

// PreviewInputTables uses SQLite to read the same initial data used for execution.
func PreviewInputTables(source string, format string) ([]InputTable, error) {
	db, cleanup, err := OpenTempSQLiteDB()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	defer db.Close()
	if err := CreateInputDatabase(db, source, format); err != nil {
		return nil, err
	}

	names, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT GLOB 'sqlite_*' ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	tables := make([]InputTable, 0)
	for names.Next() {
		var name string
		if err := names.Scan(&name); err != nil {
			names.Close()
			return nil, err
		}
		tables = append(tables, InputTable{Name: name})
	}
	err = names.Err()
	names.Close()
	if err != nil {
		return nil, err
	}

	for i := range tables {
		rows, err := db.Query("SELECT * FROM " + quoteIdentifier(tables[i].Name))
		if err != nil {
			return nil, err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}
		records, err := readRows(rows, len(columns))
		rows.Close()
		if err != nil {
			return nil, err
		}
		tables[i].Rows = append([][]string{columns}, records...)
	}
	return tables, nil
}
