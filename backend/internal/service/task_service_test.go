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

func TestTaskServiceSubmitTreatsEquivalentNumbersAsEqual(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "009.json", `{
		"number": 9,
		"title": "Average",
		"statement": "Average rows.",
		"constraints": [],
		"csv": "score\n100\n70",
		"starterSql": "SELECT AVG(CAST(score AS INTEGER)) AS average_score FROM input;",
		"solutionSql": "SELECT AVG(CAST(score AS INTEGER)) AS average_score FROM input;",
		"expectedCsv": "average_score\n85.0\n",
		"tests": [
			{
				"name": "sample",
				"csv": "score\n100\n70",
				"expectedCsv": "average_score\n85.0\n"
			}
		]
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := NewTaskService(NewSQLiteService())

	result, err := service.Submit("9", "SELECT 85 AS average_score;")
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if !result.Passed {
		t.Fatalf("Submit() passed = false, want true: %#v", result)
	}
}

func TestTaskServiceSubmitForUserRecordsSubmission(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "010.json", `{
		"number": 10,
		"slug": "select-all",
		"title": "Select all",
		"statement": "Select all rows.",
		"constraints": [],
		"csv": "id\n1",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n1\n",
		"tests": [
			{
				"name": "sample",
				"csv": "id\n1",
				"expectedCsv": "id\n1\n"
			}
		]
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := NewTaskService(NewSQLiteService())

	result, err := service.SubmitForUser("10", "SELECT id FROM input;", "user_test")
	if err != nil {
		t.Fatalf("SubmitForUser returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("SubmitForUser() passed = false, want true: %#v", result)
	}

	submissions := service.Submissions()
	if len(submissions) != 1 {
		t.Fatalf("submission count = %d, want 1", len(submissions))
	}
	if submissions[0].UserID != "user_test" {
		t.Fatalf("submission user = %q, want user_test", submissions[0].UserID)
	}
	if submissions[0].TaskNumber != 10 || submissions[0].TaskTitle != "Select all" {
		t.Fatalf("submission task = %#v, want task 10 Select all", submissions[0])
	}
}

func writeTaskFile(t *testing.T, dir string, fileName string, content string) {
	t.Helper()

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}
}
