package helper

import "testing"

func TestSplitSQLStatementsIgnoresSemicolonInString(t *testing.T) {
	got := SplitSQLStatements(`SELECT 'a;b'; SELECT "c;d"`)
	want := []string{`SELECT 'a;b'`, `SELECT "c;d"`}

	if len(got) != len(want) {
		t.Fatalf("SplitSQLStatements() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitSQLStatements()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
