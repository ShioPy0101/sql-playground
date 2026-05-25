package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Constraint struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TaskTestCase struct {
	Name        string `json:"name"`
	CSV         string `json:"csv"`
	ExpectedCSV string `json:"expectedCsv"`
}

type Task struct {
	Number      int            `json:"number"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Statement   string         `json:"statement"`
	Constraints []Constraint   `json:"constraints"`
	CSV         string         `json:"csv"`
	StarterSQL  string         `json:"starterSql"`
	SolutionSQL string         `json:"solutionSql"`
	ExpectedCSV string         `json:"expectedCsv"`
	Tests       []TaskTestCase `json:"tests"`
}

type PublicTask struct {
	Number      int          `json:"number"`
	Slug        string       `json:"slug"`
	Title       string       `json:"title"`
	Statement   string       `json:"statement"`
	Constraints []Constraint `json:"constraints"`
	CSV         string       `json:"csv"`
	StarterSQL  string       `json:"starterSql"`
	ExpectedCSV string       `json:"expectedCsv"`
	TestCount   int          `json:"testCount"`
}

type TaskSubmitResult struct {
	Passed bool             `json:"passed"`
	Cases  []TaskCaseResult `json:"cases"`
}

type TaskCaseResult struct {
	Name        string `json:"name"`
	Passed      bool   `json:"passed"`
	ActualCSV   string `json:"actualCsv"`
	ExpectedCSV string `json:"expectedCsv"`
	Error       string `json:"error,omitempty"`
}

type TaskService struct {
	sqliteService *SQLiteService
	taskDirs      []string
}

func NewTaskService(sqliteService *SQLiteService) *TaskService {
	return &TaskService{
		sqliteService: sqliteService,
		taskDirs:      taskDirectories(),
	}
}

func (s *TaskService) GetPublicTask(number string) (PublicTask, error) {
	task, err := s.LoadTask(number)
	if err != nil {
		return PublicTask{}, err
	}

	return publicTask(task), nil
}

func (s *TaskService) Submit(number string, query string) (TaskSubmitResult, error) {
	task, err := s.LoadTask(number)
	if err != nil {
		return TaskSubmitResult{}, err
	}

	tests := task.Tests
	if len(tests) == 0 {
		tests = []TaskTestCase{{
			Name:        "sample",
			CSV:         task.CSV,
			ExpectedCSV: task.ExpectedCSV,
		}}
	}

	result := TaskSubmitResult{Passed: true, Cases: make([]TaskCaseResult, 0, len(tests))}
	for _, test := range tests {
		actualCSV, err := s.sqliteService.Execute(test.CSV, query)
		caseResult := TaskCaseResult{
			Name:        test.Name,
			ExpectedCSV: test.ExpectedCSV,
		}
		if err != nil {
			caseResult.Error = err.Error()
			result.Passed = false
			result.Cases = append(result.Cases, caseResult)
			continue
		}

		caseResult.ActualCSV = actualCSV
		caseResult.Passed = equivalentCSV(actualCSV, test.ExpectedCSV)
		if !caseResult.Passed {
			result.Passed = false
		}
		result.Cases = append(result.Cases, caseResult)
	}

	return result, nil
}

func (s *TaskService) LoadTask(number string) (Task, error) {
	fileNames := taskFileNames(number)
	for _, dir := range s.taskDirs {
		for _, fileName := range fileNames {
			path := filepath.Join(dir, fileName)
			content, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return Task{}, err
			}

			var task Task
			if err := json.Unmarshal(content, &task); err != nil {
				return Task{}, fmt.Errorf("invalid task file %s: %w", path, err)
			}
			return task, nil
		}
	}

	return Task{}, fmt.Errorf("task %s was not found", number)
}

func taskFileNames(number string) []string {
	trimmed := strings.TrimSpace(number)
	names := []string{fmt.Sprintf("%s.json", trimmed)}

	var parsed int
	if _, err := fmt.Sscanf(trimmed, "%d", &parsed); err == nil {
		padded := fmt.Sprintf("%03d.json", parsed)
		if padded != names[0] {
			names = append(names, padded)
		}
	}

	return names
}

func taskDirectories() []string {
	if dir := os.Getenv("TASKS_DIR"); dir != "" {
		return []string{dir}
	}

	return []string{"tasks", "backend/tasks"}
}

func publicTask(task Task) PublicTask {
	return PublicTask{
		Number:      task.Number,
		Slug:        task.Slug,
		Title:       task.Title,
		Statement:   task.Statement,
		Constraints: task.Constraints,
		CSV:         task.CSV,
		StarterSQL:  task.StarterSQL,
		ExpectedCSV: task.ExpectedCSV,
		TestCount:   len(task.Tests),
	}
}

func equivalentCSV(left string, right string) bool {
	leftRows, leftErr := normalizedCSV(left)
	rightRows, rightErr := normalizedCSV(right)
	if leftErr != nil || rightErr != nil {
		return strings.TrimSpace(left) == strings.TrimSpace(right)
	}

	if len(leftRows) != len(rightRows) {
		return false
	}
	for rowIndex := range leftRows {
		if len(leftRows[rowIndex]) != len(rightRows[rowIndex]) {
			return false
		}
		for cellIndex := range leftRows[rowIndex] {
			if !equivalentCSVCell(leftRows[rowIndex][cellIndex], rightRows[rowIndex][cellIndex]) {
				return false
			}
		}
	}
	return true
}

func equivalentCSVCell(left string, right string) bool {
	if left == right {
		return true
	}

	leftNumber, leftErr := strconv.ParseFloat(left, 64)
	rightNumber, rightErr := strconv.ParseFloat(right, 64)
	return leftErr == nil && rightErr == nil && leftNumber == rightNumber
}

func normalizedCSV(text string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(text)))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for i := range record {
			record[i] = strings.TrimSpace(record[i])
		}
		rows = append(rows, record)
	}
	return rows, nil
}
