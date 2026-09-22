package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type SubmissionStore struct {
	db      *sql.DB
	dialect string
}

func NewSubmissionStore(dbPath string) (*SubmissionStore, error) {
	if databaseURL := submissionsDatabaseURL(); databaseURL != "" {
		return newPostgresSubmissionStore(databaseURL)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SubmissionStore{db: db, dialect: "sqlite"}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func newPostgresSubmissionStore(databaseURL string) (*SubmissionStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	store := &SubmissionStore{db: db, dialect: "postgres"}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SubmissionStore) Insert(submission TaskSubmission) (int, error) {
	casesJSON, err := json.Marshal(submission.Cases)
	if err != nil {
		return 0, err
	}

	query := `
		INSERT INTO task_submissions (
			event_id,
			user_id,
			task_number,
			task_slug,
			task_title,
			query,
			passed,
			cases_json,
			submitted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	args := []any{
		submission.EventID,
		submission.UserID,
		submission.TaskNumber,
		submission.TaskSlug,
		submission.TaskTitle,
		submission.Query,
		submission.Passed,
		string(casesJSON),
		submission.SubmittedAt.Format(time.RFC3339Nano),
	}
	if s.dialect == "postgres" {
		var id int
		err := s.db.QueryRow(s.rebind(query+" RETURNING id"), args...).Scan(&id)
		return id, err
	}
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func (s *SubmissionStore) UpdateBenchmarkReport(submissionID int, userID string, taskNumber int, report BenchmarkReport) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}
	result, err := s.db.Exec(s.rebind(`UPDATE task_submissions SET benchmark_json = ? WHERE id = ? AND user_id = ? AND task_number = ?`), string(reportJSON), submissionID, userID, taskNumber)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("submission %d was not found", submissionID)
	}
	return nil
}

func (s *SubmissionStore) List() ([]TaskSubmission, error) {
	rows, err := s.db.Query(`
		SELECT
			id,
			event_id,
			user_id,
			task_number,
			task_slug,
			task_title,
			query,
			passed,
			cases_json,
			benchmark_json,
			submitted_at
		FROM task_submissions
		ORDER BY submitted_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubmissions(rows)
}

func (s *SubmissionStore) ListByUserTask(userID string, taskNumber int, eventID *int64) ([]TaskSubmission, error) {
	whereEvent := "event_id IS NULL"
	args := []any{userID, taskNumber}
	if eventID != nil {
		whereEvent = "event_id = ?"
		args = append(args, *eventID)
	}
	rows, err := s.db.Query(s.rebind(`
		SELECT
			id,
			event_id,
			user_id,
			task_number,
			task_slug,
			task_title,
			query,
			passed,
			cases_json,
			benchmark_json,
			submitted_at
		FROM task_submissions
		WHERE user_id = ? AND task_number = ? AND `+whereEvent+`
		ORDER BY submitted_at DESC, id DESC
	`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubmissions(rows)
}

func (s *SubmissionStore) ProgressByUser(userID string) (map[int]UserTaskProgress, error) {
	rows, err := s.db.Query(s.rebind(`
		SELECT
			task_number,
			query,
			passed,
			submitted_at
		FROM task_submissions
		WHERE user_id = ? AND event_id IS NULL
		ORDER BY submitted_at DESC, id DESC
	`), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progress := map[int]UserTaskProgress{}
	for rows.Next() {
		var taskNumber int
		var query string
		var passed bool
		var submittedAt string
		if err := rows.Scan(&taskNumber, &query, &passed, &submittedAt); err != nil {
			return nil, err
		}

		parsedAt, err := time.Parse(time.RFC3339Nano, submittedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid submission timestamp for task %d: %w", taskNumber, err)
		}

		taskProgress := progress[taskNumber]
		if !taskProgress.Answered {
			taskProgress = UserTaskProgress{
				TaskNumber: taskNumber,
				Answered:   true,
				LastQuery:  query,
				UpdatedAt:  parsedAt,
			}
		}
		if passed {
			taskProgress.Solved = true
		}
		progress[taskNumber] = taskProgress
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return progress, nil
}

func scanSubmissions(rows *sql.Rows) ([]TaskSubmission, error) {
	submissions := []TaskSubmission{}
	for rows.Next() {
		var submission TaskSubmission
		var casesJSON string
		var benchmarkJSON sql.NullString
		var submittedAt string
		if err := rows.Scan(
			&submission.ID,
			&submission.EventID,
			&submission.UserID,
			&submission.TaskNumber,
			&submission.TaskSlug,
			&submission.TaskTitle,
			&submission.Query,
			&submission.Passed,
			&casesJSON,
			&benchmarkJSON,
			&submittedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(casesJSON), &submission.Cases); err != nil {
			return nil, fmt.Errorf("invalid submission cases for id %d: %w", submission.ID, err)
		}
		if benchmarkJSON.Valid && benchmarkJSON.String != "" {
			var report BenchmarkReport
			if err := json.Unmarshal([]byte(benchmarkJSON.String), &report); err != nil {
				return nil, fmt.Errorf("invalid submission benchmark for id %d: %w", submission.ID, err)
			}
			submission.BenchmarkReport = &report
		}

		parsedAt, err := time.Parse(time.RFC3339Nano, submittedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid submission timestamp for id %d: %w", submission.ID, err)
		}
		submission.SubmittedAt = parsedAt
		submissions = append(submissions, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}

func (s *SubmissionStore) migrate() error {
	if s.dialect == "postgres" {
		if err := s.migratePostgres(); err != nil {
			return err
		}
		return s.migrateEvents()
	}

	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS task_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NULL,
			user_id TEXT NOT NULL,
			task_number INTEGER NOT NULL,
			task_slug TEXT NOT NULL,
			task_title TEXT NOT NULL,
			query TEXT NOT NULL,
			passed INTEGER NOT NULL,
			cases_json TEXT NOT NULL,
			benchmark_json TEXT NULL,
			submitted_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_submitted_at
			ON task_submissions (submitted_at DESC, id DESC);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_id
			ON task_submissions (user_id);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_task
			ON task_submissions (user_id, task_number, submitted_at DESC, id DESC);
	`)
	if err != nil {
		return err
	}
	if !s.sqliteColumnExists("task_submissions", "event_id") {
		if _, err := s.db.Exec(`ALTER TABLE task_submissions ADD COLUMN event_id INTEGER NULL`); err != nil {
			return err
		}
	}
	if !s.sqliteColumnExists("task_submissions", "benchmark_json") {
		if _, err := s.db.Exec(`ALTER TABLE task_submissions ADD COLUMN benchmark_json TEXT NULL`); err != nil {
			return err
		}
	}
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_task_submissions_event
			ON task_submissions (event_id, submitted_at ASC, id ASC);
	`)
	if err != nil {
		return err
	}
	return s.migrateEvents()
}

func (s *SubmissionStore) migratePostgres() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS task_submissions (
			id BIGSERIAL PRIMARY KEY,
			event_id BIGINT NULL,
			user_id TEXT NOT NULL,
			task_number INTEGER NOT NULL,
			task_slug TEXT NOT NULL,
			task_title TEXT NOT NULL,
			query TEXT NOT NULL,
			passed BOOLEAN NOT NULL,
			cases_json TEXT NOT NULL,
			benchmark_json TEXT NULL,
			submitted_at TEXT NOT NULL
		);

		ALTER TABLE task_submissions ADD COLUMN IF NOT EXISTS event_id BIGINT NULL;
		ALTER TABLE task_submissions ADD COLUMN IF NOT EXISTS benchmark_json TEXT NULL;

		CREATE INDEX IF NOT EXISTS idx_task_submissions_submitted_at
			ON task_submissions (submitted_at DESC, id DESC);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_id
			ON task_submissions (user_id);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_task
			ON task_submissions (user_id, task_number, submitted_at DESC, id DESC);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_event
			ON task_submissions (event_id, submitted_at ASC, id ASC);
	`)
	return err
}

func (s *SubmissionStore) sqliteColumnExists(table, column string) bool {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err == nil && name == column {
			return true
		}
	}
	return false
}

func (s *SubmissionStore) rebind(query string) string {
	if s.dialect != "postgres" {
		return query
	}

	var builder strings.Builder
	placeholder := 1
	for _, char := range query {
		if char == '?' {
			builder.WriteString(fmt.Sprintf("$%d", placeholder))
			placeholder++
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func submissionsDatabaseURL() string {
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}

	return os.Getenv("POSTGRES_URL")
}
