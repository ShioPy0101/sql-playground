package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
	SavedQuery  string       `json:"savedQuery,omitempty"`
	Answered    bool         `json:"answered"`
	Solved      bool         `json:"solved"`
}

type PublicTaskSummary struct {
	Number    int        `json:"number"`
	Slug      string     `json:"slug"`
	Title     string     `json:"title"`
	Statement string     `json:"statement"`
	TestCount int        `json:"testCount"`
	Answered  bool       `json:"answered"`
	Solved    bool       `json:"solved"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
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
	submissions   *SubmissionStore
}

type TaskSubmission struct {
	ID          int              `json:"id"`
	UserID      string           `json:"userId"`
	TaskNumber  int              `json:"taskNumber"`
	TaskSlug    string           `json:"taskSlug"`
	TaskTitle   string           `json:"taskTitle"`
	Query       string           `json:"query"`
	Passed      bool             `json:"passed"`
	Cases       []TaskCaseResult `json:"cases"`
	SubmittedAt time.Time        `json:"submittedAt"`
}

type UserTaskProgress struct {
	TaskNumber int
	Answered   bool
	Solved     bool
	LastQuery  string
	UpdatedAt  time.Time
}

func NewTaskService(sqliteService *SQLiteService) (*TaskService, error) {
	return NewTaskServiceWithSubmissionDB(sqliteService, submissionsDBPath())
}

func NewTaskServiceWithSubmissionDB(sqliteService *SQLiteService, dbPath string) (*TaskService, error) {
	submissions, err := NewSubmissionStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &TaskService{
		sqliteService: sqliteService,
		taskDirs:      taskDirectories(),
		submissions:   submissions,
	}, nil
}

func (s *TaskService) GetPublicTask(number string) (PublicTask, error) {
	return s.GetPublicTaskForUser(number, "")
}

func (s *TaskService) GetPublicTaskForUser(number string, userID string) (PublicTask, error) {
	task, err := s.LoadTask(number)
	if err != nil {
		return PublicTask{}, err
	}

	public := publicTask(task)
	if userID == "" {
		return public, nil
	}

	progress, err := s.submissions.ProgressByUser(userID)
	if err != nil {
		return PublicTask{}, err
	}
	applyProgressToTask(&public, progress[task.Number])

	return public, nil
}

func (s *TaskService) ListPublicTasks() ([]PublicTaskSummary, error) {
	return s.ListPublicTasksForUser("")
}

func (s *TaskService) ListPublicTasksForUser(userID string) ([]PublicTaskSummary, error) {
	summaries := []PublicTaskSummary{}
	seen := map[int]bool{}
	progress := map[int]UserTaskProgress{}

	if userID != "" {
		var err error
		progress, err = s.submissions.ProgressByUser(userID)
		if err != nil {
			return nil, err
		}
	}

	for _, dir := range s.taskDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}

			content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}

			var task Task
			if err := json.Unmarshal(content, &task); err != nil {
				return nil, fmt.Errorf("invalid task file %s: %w", filepath.Join(dir, entry.Name()), err)
			}
			if seen[task.Number] {
				continue
			}

			summary := publicTaskSummary(task)
			applyProgressToSummary(&summary, progress[task.Number])
			summaries = append(summaries, summary)
			seen[task.Number] = true
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Number < summaries[j].Number
	})

	return summaries, nil
}

func (s *TaskService) Submit(number string, query string) (TaskSubmitResult, error) {
	return s.submit(number, query, "")
}

func (s *TaskService) SubmitForUser(number string, query string, userID string) (TaskSubmitResult, error) {
	return s.submit(number, query, userID)
}

func (s *TaskService) Submissions() ([]TaskSubmission, error) {
	return s.submissions.List()
}

func (s *TaskService) submit(number string, query string, userID string) (TaskSubmitResult, error) {
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

	if userID != "" {
		if err := s.recordSubmission(task, query, result, userID); err != nil {
			return TaskSubmitResult{}, err
		}
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

func submissionsDBPath() string {
	if path := os.Getenv("SUBMISSIONS_DB_PATH"); path != "" {
		return path
	}

	if os.Getenv("VERCEL") != "" {
		return filepath.Join(os.TempDir(), "submissions.sqlite")
	}

	return filepath.Join("data", "submissions.sqlite")
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

func publicTaskSummary(task Task) PublicTaskSummary {
	return PublicTaskSummary{
		Number:    task.Number,
		Slug:      task.Slug,
		Title:     task.Title,
		Statement: task.Statement,
		TestCount: len(task.Tests),
	}
}

func applyProgressToTask(task *PublicTask, progress UserTaskProgress) {
	task.Answered = progress.Answered
	task.Solved = progress.Solved
	task.SavedQuery = progress.LastQuery
}

func applyProgressToSummary(summary *PublicTaskSummary, progress UserTaskProgress) {
	summary.Answered = progress.Answered
	summary.Solved = progress.Solved
	if !progress.UpdatedAt.IsZero() {
		summary.UpdatedAt = &progress.UpdatedAt
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

func (s *TaskService) recordSubmission(task Task, query string, result TaskSubmitResult, userID string) error {
	submission := TaskSubmission{
		UserID:      userID,
		TaskNumber:  task.Number,
		TaskSlug:    task.Slug,
		TaskTitle:   task.Title,
		Query:       query,
		Passed:      result.Passed,
		Cases:       append([]TaskCaseResult(nil), result.Cases...),
		SubmittedAt: time.Now().UTC(),
	}
	return s.submissions.Insert(submission)
}
