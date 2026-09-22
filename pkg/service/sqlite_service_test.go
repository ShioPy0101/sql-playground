package service

import (
	"strings"
	"testing"
)

func TestSQLiteServiceExecuteSelect(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		"id,name\n1,Ada\n2,Linus\n",
		`SELECT name FROM input WHERE id = '1'`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := "name\nAda\n"
	if got != want {
		t.Fatalf("Execute() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteMultipleQueriesAddsTraceHeaders(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		"id,name\n1,Ada\n2,Linus\n",
		`
			CREATE TABLE names AS SELECT name FROM input ORDER BY id;
			SELECT name FROM names;
			SELECT count(*) AS total FROM names;
		`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	for _, part := range []string{
		"-- Query 1: CREATE TABLE names AS SELECT name FROM input ORDER BY id",
		"OK\n",
		"-- Query 2: SELECT name FROM names\nname\nAda\nLinus\n",
		"-- Query 3: SELECT count(*) AS total FROM names\ntotal\n2\n",
	} {
		if !strings.Contains(got, part) {
			t.Fatalf("Execute() output did not contain %q:\n%s", part, got)
		}
	}
}

func TestSQLiteServiceExecuteMultipleQueriesCompactsTraceHeaders(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		"id,name\n1,Ada\n2,Linus\n",
		`
			SELECT
				name
			FROM input
			ORDER BY id;
			SELECT count(*) AS total FROM input;
		`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !strings.Contains(got, "-- Query 1: SELECT name FROM input ORDER BY id") {
		t.Fatalf("Execute() output did not compact query header:\n%s", got)
	}
}

func TestSQLiteServiceExecuteMultipleTables(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		`# table: users
id,name
1,Ada
2,Linus

# table: orders
id,user_id,total
100,1,48
101,1,52
102,2,37
`,
		`
			SELECT users.name, SUM(orders.total) AS total
			FROM users
			JOIN orders ON orders.user_id = users.id
			GROUP BY users.id, users.name
			ORDER BY total DESC;
		`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := "name,total\nAda,100\nLinus,37\n"
	if got != want {
		t.Fatalf("Execute() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteInfersIntegerColumns(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		`[orders]
id,total
1,9
2,100
10,20
`,
		`
			SELECT id, total
			FROM orders
			ORDER BY total DESC, id ASC;
		`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := "id,total\n2,100\n10,20\n1,9\n"
	if got != want {
		t.Fatalf("Execute() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteKeepsLeadingZeroColumnsAsText(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		`code
001
010
`,
		`SELECT code FROM input ORDER BY code DESC;`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := "code\n010\n001\n"
	if got != want {
		t.Fatalf("Execute() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteMultipleTablesWithBracketMarkers(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.Execute(
		`[users]
id,name
1,Ada

[teams]
user_id,team
1,core
`,
		`SELECT users.name, teams.team FROM users JOIN teams ON teams.user_id = users.id`,
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	want := "name,team\nAda,core\n"
	if got != want {
		t.Fatalf("Execute() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteAndInspect(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.ExecuteAndInspect(
		"",
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`,
		`SELECT name, type, "notnull", pk FROM pragma_table_info('users') ORDER BY cid;`,
	)
	if err != nil {
		t.Fatalf("ExecuteAndInspect returned error: %v", err)
	}

	want := "name,type,notnull,pk\nid,INTEGER,0,1\nname,TEXT,1,0\n"
	if got != want {
		t.Fatalf("ExecuteAndInspect() = %q, want %q", got, want)
	}
}

func TestSQLiteServiceExecuteSQLInput(t *testing.T) {
	service := NewSQLiteService()

	got, err := service.ExecuteInput(
		`CREATE TABLE scores (name TEXT PRIMARY KEY, score INTEGER UNIQUE);
		 INSERT INTO scores (name, score) VALUES ('Ada', 82), ('Linus', 91);`,
		"sql",
		`SELECT name FROM scores ORDER BY score DESC;`,
	)
	if err != nil {
		t.Fatalf("ExecuteInput returned error: %v", err)
	}

	want := "name\nLinus\nAda\n"
	if got != want {
		t.Fatalf("ExecuteInput() = %q, want %q", got, want)
	}
}
