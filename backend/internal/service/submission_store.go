package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SubmissionStore struct {
	db *sql.DB
}

func NewSubmissionStore(dbPath string) (*SubmissionStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SubmissionStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SubmissionStore) Insert(submission TaskSubmission) error {
	casesJSON, err := json.Marshal(submission.Cases)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO task_submissions (
			user_id,
			task_number,
			task_slug,
			task_title,
			query,
			passed,
			cases_json,
			submitted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		submission.UserID,
		submission.TaskNumber,
		submission.TaskSlug,
		submission.TaskTitle,
		submission.Query,
		submission.Passed,
		string(casesJSON),
		submission.SubmittedAt.Format(time.RFC3339Nano),
	)
	return err
}

func (s *SubmissionStore) List() ([]TaskSubmission, error) {
	rows, err := s.db.Query(`
		SELECT
			id,
			user_id,
			task_number,
			task_slug,
			task_title,
			query,
			passed,
			cases_json,
			submitted_at
		FROM task_submissions
		ORDER BY submitted_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []TaskSubmission
	for rows.Next() {
		var submission TaskSubmission
		var casesJSON string
		var submittedAt string
		if err := rows.Scan(
			&submission.ID,
			&submission.UserID,
			&submission.TaskNumber,
			&submission.TaskSlug,
			&submission.TaskTitle,
			&submission.Query,
			&submission.Passed,
			&casesJSON,
			&submittedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(casesJSON), &submission.Cases); err != nil {
			return nil, fmt.Errorf("invalid submission cases for id %d: %w", submission.ID, err)
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

func (s *SubmissionStore) ProgressByUser(userID string) (map[int]UserTaskProgress, error) {
	rows, err := s.db.Query(`
		SELECT
			task_number,
			query,
			passed,
			submitted_at
		FROM task_submissions
		WHERE user_id = ?
		ORDER BY submitted_at DESC, id DESC
	`, userID)
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

func (s *SubmissionStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS task_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			task_number INTEGER NOT NULL,
			task_slug TEXT NOT NULL,
			task_title TEXT NOT NULL,
			query TEXT NOT NULL,
			passed INTEGER NOT NULL,
			cases_json TEXT NOT NULL,
			submitted_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_submitted_at
			ON task_submissions (submitted_at DESC, id DESC);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_id
			ON task_submissions (user_id);

		CREATE INDEX IF NOT EXISTS idx_task_submissions_user_task
			ON task_submissions (user_id, task_number, submitted_at DESC, id DESC);
	`)
	return err
}
