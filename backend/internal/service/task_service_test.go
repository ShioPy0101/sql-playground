package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTaskServiceSubmitComparesAllCases(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "007.json", `{
		"number": 7,
		"title": "Count rows",
		"statement": "Count input rows.",
		"constraints": [],
		"csv": "id\n1\n2",
		"starterSql": "SELECT COUNT(*) AS total FROM input;",
		"solutionSql": "SELECT COUNT(*) AS total FROM input;",
		"expectedCsv": "total\n2\n",
		"tests": [
			{
				"name": "sample",
				"csv": "id\n1\n2",
				"expectedCsv": "total\n2\n"
			},
			{
				"name": "hidden",
				"csv": "id\n1\n2\n3",
				"expectedCsv": "total\n3\n"
			}
		]
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := NewTaskService(NewSQLiteService())

	result, err := service.Submit("7", "SELECT COUNT(*) AS total FROM input;")
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if !result.Passed {
		t.Fatalf("Submit() passed = false, want true: %#v", result)
	}
	if len(result.Cases) != 2 {
		t.Fatalf("Submit() case count = %d, want 2", len(result.Cases))
	}
}

func TestTaskServiceSubmitReportsWrongAnswer(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "008.json", `{
		"number": 8,
		"title": "Select rows",
		"statement": "Select rows.",
		"constraints": [],
		"csv": "id\n1\n2",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input ORDER BY id;",
		"expectedCsv": "id\n1\n2\n",
		"tests": [
			{
				"name": "sample",
				"csv": "id\n1\n2",
				"expectedCsv": "id\n1\n2\n"
			}
		]
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := NewTaskService(NewSQLiteService())

	result, err := service.Submit("008", "SELECT id FROM input WHERE id = '1';")
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if result.Passed {
		t.Fatalf("Submit() passed = true, want false")
	}
	if result.Cases[0].Passed {
		t.Fatalf("Submit() first case passed = true, want false")
	}
}

func writeTaskFile(t *testing.T, dir string, fileName string, content string) {
	t.Helper()

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}
}
