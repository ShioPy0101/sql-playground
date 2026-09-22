package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShioPy0101/sql-playground/pkg/service/helper"
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
	service := newTestTaskService(t)

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

func TestTaskServiceSubmitNeverRunsConfiguredBenchmark(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "906.json", `{
		"number": 906,
		"mode": "sql",
		"benchmark": {
			"enabled": true,
			"target": "submission",
			"rowCounts": [1000000]
		},
		"title": "Separated benchmark",
		"statement": "Submit must only grade.",
		"constraints": [],
		"csv": "id\n1",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n1\n",
		"tests": [{
			"name": "sample",
			"csv": "id\n1",
			"expectedCsv": "id\n1\n"
		}]
	}`)
	t.Setenv("TASKS_DIR", taskDir)
	t.Setenv("SQLITE_BENCHMARK_MAX_ROWS", "1")
	service := newTestTaskService(t)

	result, err := service.Submit("906", "SELECT id FROM input;")
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("Submit() passed = false, want true: %#v", result)
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
	service := newTestTaskService(t)

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
	service := newTestTaskService(t)

	result, err := service.Submit("9", "SELECT 85 AS average_score;")
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}

	if !result.Passed {
		t.Fatalf("Submit() passed = false, want true: %#v", result)
	}
}

func TestTaskServiceSubmitChecksDDLSchema(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "011.json", `{
		"number": 11,
		"title": "Create users table",
		"statement": "Create a users table.",
		"constraints": [],
		"csv": "",
		"starterSql": "CREATE TABLE users ();",
		"solutionSql": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);",
		"checkSql": "SELECT name, type, \"notnull\", pk FROM pragma_table_info('users') ORDER BY cid;",
		"expectedCsv": "name,type,notnull,pk\nid,INTEGER,0,1\nname,TEXT,1,0\n",
		"tests": []
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := newTestTaskService(t)

	result, err := service.Submit("11", "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);")
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
	dbPath := filepath.Join(t.TempDir(), "submissions.sqlite")
	service, err := NewTaskServiceWithSubmissionDB(NewSQLiteService(), dbPath)
	if err != nil {
		t.Fatalf("failed to create task service: %v", err)
	}

	result, err := service.SubmitForUser("10", "SELECT id FROM input;", "user_test")
	if err != nil {
		t.Fatalf("SubmitForUser returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("SubmitForUser() passed = false, want true: %#v", result)
	}

	submissions, err := service.Submissions()
	if err != nil {
		t.Fatalf("Submissions returned error: %v", err)
	}
	if len(submissions) != 1 {
		t.Fatalf("submission count = %d, want 1", len(submissions))
	}
	if submissions[0].UserID != "user_test" {
		t.Fatalf("submission user = %q, want user_test", submissions[0].UserID)
	}
	if submissions[0].TaskNumber != 10 || submissions[0].TaskTitle != "Select all" {
		t.Fatalf("submission task = %#v, want task 10 Select all", submissions[0])
	}

	reopened, err := NewTaskServiceWithSubmissionDB(NewSQLiteService(), dbPath)
	if err != nil {
		t.Fatalf("failed to reopen task service: %v", err)
	}
	persisted, err := reopened.Submissions()
	if err != nil {
		t.Fatalf("reopened Submissions returned error: %v", err)
	}
	if len(persisted) != 1 || persisted[0].UserID != "user_test" {
		t.Fatalf("persisted submissions = %#v, want one user_test submission", persisted)
	}
}

func TestTaskServiceSubmissionsForUserTaskFiltersHistory(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "001.json", `{
		"number": 1,
		"slug": "first",
		"title": "First",
		"statement": "First task.",
		"constraints": [],
		"csv": "id\n1",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n1\n",
		"tests": []
	}`)
	writeTaskFile(t, taskDir, "002.json", `{
		"number": 2,
		"slug": "second",
		"title": "Second",
		"statement": "Second task.",
		"constraints": [],
		"csv": "id\n2",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n2\n",
		"tests": []
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := newTestTaskService(t)

	if _, err := service.SubmitForUser("1", "SELECT id FROM input;", "user_a"); err != nil {
		t.Fatalf("SubmitForUser user_a task 1 returned error: %v", err)
	}
	if _, err := service.SubmitForUser("1", "SELECT id FROM input WHERE id = '1';", "user_a"); err != nil {
		t.Fatalf("second SubmitForUser user_a task 1 returned error: %v", err)
	}
	if _, err := service.SubmitForUser("2", "SELECT id FROM input;", "user_a"); err != nil {
		t.Fatalf("SubmitForUser user_a task 2 returned error: %v", err)
	}
	if _, err := service.SubmitForUser("1", "SELECT id FROM input;", "user_b"); err != nil {
		t.Fatalf("SubmitForUser user_b task 1 returned error: %v", err)
	}

	history, err := service.SubmissionsForUserTask("1", "user_a")
	if err != nil {
		t.Fatalf("SubmissionsForUserTask returned error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history count = %d, want 2", len(history))
	}
	for _, submission := range history {
		if submission.UserID != "user_a" || submission.TaskNumber != 1 {
			t.Fatalf("history included unrelated submission: %#v", submission)
		}
	}
}

func TestTaskServiceCheckAllSolutions(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "001.json", `{
		"number": 1,
		"slug": "passing",
		"title": "Passing",
		"statement": "Passing task.",
		"constraints": [],
		"csv": "id\n1",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n1\n",
		"tests": []
	}`)
	writeTaskFile(t, taskDir, "002.json", `{
		"number": 2,
		"slug": "failing",
		"title": "Failing",
		"statement": "Failing task.",
		"constraints": [],
		"csv": "id\n2",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input WHERE id = 'missing';",
		"expectedCsv": "id\n2\n",
		"tests": []
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := newTestTaskService(t)

	checks, err := service.CheckAllSolutions()
	if err != nil {
		t.Fatalf("CheckAllSolutions returned error: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("check count = %d, want 2", len(checks))
	}
	if !checks[0].Passed {
		t.Fatalf("first check passed = false, want true: %#v", checks[0])
	}
	if checks[1].Passed {
		t.Fatalf("second check passed = true, want false")
	}
}

func TestTaskServiceSubmissionsReturnsEmptySlice(t *testing.T) {
	service := newTestTaskService(t)

	submissions, err := service.Submissions()
	if err != nil {
		t.Fatalf("Submissions returned error: %v", err)
	}
	if submissions == nil {
		t.Fatalf("Submissions returned nil, want empty slice")
	}
	if len(submissions) != 0 {
		t.Fatalf("submission count = %d, want 0", len(submissions))
	}
}

func TestTaskServiceListPublicTasks(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "002.json", `{
		"number": 2,
		"slug": "second",
		"title": "Second task",
		"statement": "Second statement.",
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
	writeTaskFile(t, taskDir, "001.json", `{
		"number": 1,
		"slug": "first",
		"title": "First task",
		"statement": "First statement.",
		"constraints": [],
		"csv": "id\n1",
		"starterSql": "SELECT id FROM input;",
		"solutionSql": "SELECT id FROM input;",
		"expectedCsv": "id\n1\n",
		"tests": []
	}`)

	t.Setenv("TASKS_DIR", taskDir)
	service := newTestTaskService(t)

	tasks, err := service.ListPublicTasks()
	if err != nil {
		t.Fatalf("ListPublicTasks returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("task count = %d, want 2", len(tasks))
	}
	if tasks[0].Number != 1 || tasks[0].Slug != "first" || tasks[0].Title != "First task" {
		t.Fatalf("first task = %#v, want task 1", tasks[0])
	}
	if tasks[1].Number != 2 || tasks[1].TestCount != 1 {
		t.Fatalf("second task = %#v, want task 2 with one test", tasks[1])
	}
}

func TestTaskServiceListBundledTasks(t *testing.T) {
	service := newTestTaskService(t)

	tasks, err := service.ListPublicTasks()
	if err != nil {
		t.Fatalf("ListPublicTasks returned error: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatalf("task count = 0, want bundled tasks")
	}
	if tasks[0].Number != 1 {
		t.Fatalf("first bundled task number = %d, want 1", tasks[0].Number)
	}
}

func TestTaskServiceAppliesUserProgress(t *testing.T) {
	taskDir := t.TempDir()
	writeTaskFile(t, taskDir, "001.json", `{
		"number": 1,
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
	service := newTestTaskService(t)

	wrongQuery := "SELECT id FROM input WHERE id = '2';"
	if _, err := service.SubmitForUser("1", wrongQuery, "user_progress"); err != nil {
		t.Fatalf("wrong SubmitForUser returned error: %v", err)
	}

	tasks, err := service.ListPublicTasksForUser("user_progress")
	if err != nil {
		t.Fatalf("ListPublicTasksForUser returned error: %v", err)
	}
	if len(tasks) != 1 || !tasks[0].Answered || tasks[0].Solved {
		t.Fatalf("task progress after wrong answer = %#v, want answered and unsolved", tasks)
	}

	task, err := service.GetPublicTaskForUser("1", "user_progress")
	if err != nil {
		t.Fatalf("GetPublicTaskForUser returned error: %v", err)
	}
	if task.SavedQuery != wrongQuery {
		t.Fatalf("saved query = %q, want %q", task.SavedQuery, wrongQuery)
	}

	correctQuery := "SELECT id FROM input;"
	if _, err := service.SubmitForUser("1", correctQuery, "user_progress"); err != nil {
		t.Fatalf("correct SubmitForUser returned error: %v", err)
	}

	tasks, err = service.ListPublicTasksForUser("user_progress")
	if err != nil {
		t.Fatalf("ListPublicTasksForUser returned error: %v", err)
	}
	if !tasks[0].Answered || !tasks[0].Solved {
		t.Fatalf("task progress after correct answer = %#v, want answered and solved", tasks[0])
	}

	task, err = service.GetPublicTaskForUser("1", "user_progress")
	if err != nil {
		t.Fatalf("GetPublicTaskForUser returned error: %v", err)
	}
	if task.SavedQuery != correctQuery {
		t.Fatalf("saved query = %q, want latest query %q", task.SavedQuery, correctQuery)
	}
}

func TestTaskServiceNewAdvancedTaskSolutionsPass(t *testing.T) {
	service := newTestTaskService(t)

	for number := 22; number <= 31; number++ {
		task, err := service.LoadTask(fmt.Sprintf("%03d", number))
		if err != nil {
			t.Fatalf("LoadTask(%03d) returned error: %v", number, err)
		}

		result, err := service.Submit(fmt.Sprintf("%03d", number), task.SolutionSQL)
		if err != nil {
			t.Fatalf("Submit(%03d) returned error: %v", number, err)
		}
		if !result.Passed {
			t.Fatalf("solution for task %03d did not pass: %#v", number, result)
		}
	}
}

func TestTaskServiceTableDesignTaskSolutionsPass(t *testing.T) {
	service := newTestTaskService(t)

	for number := 42; number <= 44; number++ {
		task, err := service.LoadTask(fmt.Sprintf("%03d", number))
		if err != nil {
			t.Fatalf("LoadTask(%03d) returned error: %v", number, err)
		}

		result, err := service.Submit(fmt.Sprintf("%03d", number), task.SolutionSQL)
		if err != nil {
			t.Fatalf("Submit(%03d) returned error: %v", number, err)
		}
		if !result.Passed {
			t.Fatalf("solution for task %03d did not pass: %#v", number, result)
		}
	}
}

func TestTaskServiceBundledTaskSolutionsPass(t *testing.T) {
	service := newTestTaskService(t)

	tasks, err := service.ListPublicTasks()
	if err != nil {
		t.Fatalf("ListPublicTasks returned error: %v", err)
	}

	for _, publicTask := range tasks {
		taskNumber := fmt.Sprintf("%03d", publicTask.Number)
		task, err := service.LoadTask(taskNumber)
		if err != nil {
			t.Fatalf("LoadTask(%s) returned error: %v", taskNumber, err)
		}
		if task.SolutionSQL == "" {
			continue
		}

		result, err := service.Submit(taskNumber, task.SolutionSQL)
		if err != nil {
			t.Fatalf("Submit(%s) returned error: %v", taskNumber, err)
		}
		if !result.Passed {
			t.Fatalf("solution for task %s did not pass: %#v", taskNumber, result)
		}
	}
}

func TestTask53SolutionRejectsCrossTenantReference(t *testing.T) {
	task, err := LoadTaskDefinition("53")
	if err != nil {
		t.Fatalf("LoadTaskDefinition returned error: %v", err)
	}
	db, cleanup, err := helper.OpenTempSQLiteDB()
	if err != nil {
		t.Fatalf("OpenTempSQLiteDB returned error: %v", err)
	}
	defer cleanup()
	defer db.Close()

	if _, err := db.Exec(task.SolutionSQL); err != nil {
		t.Fatalf("execute task solution: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (tenant_id, id, name) VALUES (1, 1, 'tenant-1'), (2, 2, 'tenant-2')`); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, tenant_id, user_id, body) VALUES (1, 1, 2, 'cross tenant')`); err == nil {
		t.Fatal("cross-tenant post was accepted, want foreign-key failure")
	}
}

func TestTask50UsesTypedSQLInputWithoutCast(t *testing.T) {
	task, err := LoadTaskDefinition("50")
	if err != nil {
		t.Fatalf("LoadTaskDefinition returned error: %v", err)
	}
	if task.InputType != helper.InputFormatSQL || strings.Contains(strings.ToUpper(task.SolutionSQL), "CAST(") {
		t.Fatalf("task 50 inputType = %q, solution = %q", task.InputType, task.SolutionSQL)
	}

	service := newTestTaskService(t)
	result, err := service.Submit("50", task.SolutionSQL)
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("typed SQL input solution did not pass: %#v", result)
	}
}

func newTestTaskService(t *testing.T) *TaskService {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_URL", "")

	service, err := NewTaskServiceWithSubmissionDB(NewSQLiteService(), filepath.Join(t.TempDir(), "submissions.sqlite"))
	if err != nil {
		t.Fatalf("failed to create task service: %v", err)
	}
	return service
}

func writeTaskFile(t *testing.T, dir string, fileName string, content string) {
	t.Helper()

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}
}
