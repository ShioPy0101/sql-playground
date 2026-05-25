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
			SELECT users.name, SUM(CAST(orders.total AS INTEGER)) AS total
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

func TestSplitSQLStatementsIgnoresSemicolonInString(t *testing.T) {
	got := splitSQLStatements(`SELECT 'a;b'; SELECT "c;d"`)
	want := []string{`SELECT 'a;b'`, `SELECT "c;d"`}

	if len(got) != len(want) {
		t.Fatalf("splitSQLStatements() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitSQLStatements()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
