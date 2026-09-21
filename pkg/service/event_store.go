package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Event struct {
	ID        int64      `json:"id"`
	Slug      string     `json:"slug"`
	Title     string     `json:"title"`
	StartsAt  time.Time  `json:"startsAt"`
	EndsAt    *time.Time `json:"endsAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type EventParticipant struct {
	ID       int64     `json:"id"`
	EventID  int64     `json:"eventId"`
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joinedAt"`
}

type EventSummary struct {
	Event
	ParticipantCount int `json:"participantCount"`
	SubmissionCount  int `json:"submissionCount"`
}

type EventSubmission struct {
	TaskSubmission
	Username string `json:"username"`
}

func (s *SubmissionStore) migrateEvents() error {
	if s.dialect == "postgres" {
		_, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS events (
				id BIGSERIAL PRIMARY KEY,
				slug TEXT NOT NULL UNIQUE,
				title TEXT NOT NULL,
				starts_at TEXT NOT NULL,
				ends_at TEXT NULL,
				created_at TEXT NOT NULL
			);
			CREATE TABLE IF NOT EXISTS event_participants (
				id BIGSERIAL PRIMARY KEY,
				event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
				user_id TEXT NOT NULL,
				username TEXT NOT NULL,
				joined_at TEXT NOT NULL,
				UNIQUE (event_id, user_id)
			);
			CREATE TABLE IF NOT EXISTS event_tasks (
				event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
				task_number INTEGER NOT NULL,
				position INTEGER NOT NULL,
				PRIMARY KEY (event_id, task_number),
				UNIQUE (event_id, position)
			);
			CREATE INDEX IF NOT EXISTS idx_event_participants_event ON event_participants (event_id);
			CREATE INDEX IF NOT EXISTS idx_event_tasks_event_position ON event_tasks (event_id, position);
		`)
		return err
	}

	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			starts_at TEXT NOT NULL,
			ends_at TEXT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS event_participants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			joined_at TEXT NOT NULL,
			UNIQUE (event_id, user_id)
		);
		CREATE TABLE IF NOT EXISTS event_tasks (
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			task_number INTEGER NOT NULL,
			position INTEGER NOT NULL,
			PRIMARY KEY (event_id, task_number),
			UNIQUE (event_id, position)
		);
		CREATE INDEX IF NOT EXISTS idx_event_participants_event ON event_participants (event_id);
		CREATE INDEX IF NOT EXISTS idx_event_tasks_event_position ON event_tasks (event_id, position);
	`)
	return err
}

func (s *SubmissionStore) CreateEvent(event Event, taskNumbers []int) (Event, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Event{}, err
	}
	defer tx.Rollback()

	var endsAt any
	if event.EndsAt != nil {
		endsAt = event.EndsAt.Format(time.RFC3339Nano)
	}
	createdAt := event.CreatedAt.Format(time.RFC3339Nano)
	startsAt := event.StartsAt.Format(time.RFC3339Nano)
	if s.dialect == "postgres" {
		err = tx.QueryRow(`INSERT INTO events (slug, title, starts_at, ends_at, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`, event.Slug, event.Title, startsAt, endsAt, createdAt).Scan(&event.ID)
	} else {
		var result sql.Result
		result, err = tx.Exec(`INSERT INTO events (slug, title, starts_at, ends_at, created_at) VALUES (?, ?, ?, ?, ?)`, event.Slug, event.Title, startsAt, endsAt, createdAt)
		if err == nil {
			event.ID, err = result.LastInsertId()
		}
	}
	if err != nil {
		return Event{}, err
	}

	for index, number := range taskNumbers {
		_, err = tx.Exec(s.rebind(`INSERT INTO event_tasks (event_id, task_number, position) VALUES (?, ?, ?)`), event.ID, number, index+1)
		if err != nil {
			return Event{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func (s *SubmissionStore) EventBySlug(slug string) (Event, error) {
	var event Event
	var startsAt, createdAt string
	var endsAt sql.NullString
	err := s.db.QueryRow(s.rebind(`SELECT id, slug, title, starts_at, ends_at, created_at FROM events WHERE slug = ?`), slug).Scan(&event.ID, &event.Slug, &event.Title, &startsAt, &endsAt, &createdAt)
	if err != nil {
		return Event{}, err
	}
	if event.StartsAt, err = time.Parse(time.RFC3339Nano, startsAt); err != nil {
		return Event{}, err
	}
	if event.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Event{}, err
	}
	if endsAt.Valid {
		parsed, parseErr := time.Parse(time.RFC3339Nano, endsAt.String)
		if parseErr != nil {
			return Event{}, parseErr
		}
		event.EndsAt = &parsed
	}
	return event, nil
}

func (s *SubmissionStore) EventTaskNumbers(eventID int64) ([]int, error) {
	rows, err := s.db.Query(s.rebind(`SELECT task_number FROM event_tasks WHERE event_id = ? ORDER BY position ASC`), eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	numbers := []int{}
	for rows.Next() {
		var number int
		if err := rows.Scan(&number); err != nil {
			return nil, err
		}
		numbers = append(numbers, number)
	}
	return numbers, rows.Err()
}

func (s *SubmissionStore) EventHasTask(eventID int64, taskNumber int) (bool, error) {
	var exists int
	err := s.db.QueryRow(s.rebind(`SELECT 1 FROM event_tasks WHERE event_id = ? AND task_number = ?`), eventID, taskNumber).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *SubmissionStore) Participant(eventID int64, userID string) (*EventParticipant, error) {
	var participant EventParticipant
	var joinedAt string
	err := s.db.QueryRow(s.rebind(`SELECT id, event_id, user_id, username, joined_at FROM event_participants WHERE event_id = ? AND user_id = ?`), eventID, userID).Scan(&participant.ID, &participant.EventID, &participant.UserID, &participant.Username, &joinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	participant.JoinedAt, err = time.Parse(time.RFC3339Nano, joinedAt)
	return &participant, err
}

func (s *SubmissionStore) JoinEvent(eventID int64, userID, username string) (*EventParticipant, error) {
	now := time.Now().UTC()
	if s.dialect == "postgres" {
		_, err := s.db.Exec(`INSERT INTO event_participants (event_id, user_id, username, joined_at) VALUES ($1, $2, $3, $4) ON CONFLICT (event_id, user_id) DO UPDATE SET username = EXCLUDED.username`, eventID, userID, username, now.Format(time.RFC3339Nano))
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(`INSERT INTO event_participants (event_id, user_id, username, joined_at) VALUES (?, ?, ?, ?) ON CONFLICT(event_id, user_id) DO UPDATE SET username = excluded.username`, eventID, userID, username, now.Format(time.RFC3339Nano))
		if err != nil {
			return nil, err
		}
	}
	return s.Participant(eventID, userID)
}

func (s *SubmissionStore) ProgressByUserEvent(userID string, eventID int64) (map[int]UserTaskProgress, error) {
	rows, err := s.db.Query(s.rebind(`SELECT task_number, query, passed, submitted_at FROM task_submissions WHERE user_id = ? AND event_id = ? ORDER BY submitted_at DESC, id DESC`), userID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	progress := map[int]UserTaskProgress{}
	for rows.Next() {
		var number int
		var query, submittedAt string
		var passed bool
		if err := rows.Scan(&number, &query, &passed, &submittedAt); err != nil {
			return nil, err
		}
		parsedAt, err := time.Parse(time.RFC3339Nano, submittedAt)
		if err != nil {
			return nil, err
		}
		item := progress[number]
		if !item.Answered {
			item = UserTaskProgress{TaskNumber: number, Answered: true, LastQuery: query, UpdatedAt: parsedAt}
		}
		item.Solved = item.Solved || passed
		progress[number] = item
	}
	return progress, rows.Err()
}

func (s *SubmissionStore) ListEvents() ([]EventSummary, error) {
	rows, err := s.db.Query(`SELECT e.id, e.slug, e.title, e.starts_at, e.ends_at, e.created_at, (SELECT COUNT(*) FROM event_participants p WHERE p.event_id = e.id), (SELECT COUNT(*) FROM task_submissions s WHERE s.event_id = e.id) FROM events e ORDER BY e.starts_at DESC, e.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EventSummary{}
	for rows.Next() {
		var item EventSummary
		var startsAt, createdAt string
		var endsAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &startsAt, &endsAt, &createdAt, &item.ParticipantCount, &item.SubmissionCount); err != nil {
			return nil, err
		}
		var err error
		if item.StartsAt, err = time.Parse(time.RFC3339Nano, startsAt); err != nil {
			return nil, err
		}
		if item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
			return nil, err
		}
		if endsAt.Valid {
			parsed, err := time.Parse(time.RFC3339Nano, endsAt.String)
			if err != nil {
				return nil, err
			}
			item.EndsAt = &parsed
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SubmissionStore) EventParticipants(eventID int64) ([]EventParticipant, error) {
	rows, err := s.db.Query(s.rebind(`SELECT id, event_id, user_id, username, joined_at FROM event_participants WHERE event_id = ? ORDER BY joined_at ASC, id ASC`), eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EventParticipant{}
	for rows.Next() {
		var item EventParticipant
		var joinedAt string
		if err := rows.Scan(&item.ID, &item.EventID, &item.UserID, &item.Username, &joinedAt); err != nil {
			return nil, err
		}
		item.JoinedAt, err = time.Parse(time.RFC3339Nano, joinedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SubmissionStore) EventSubmissions(eventID int64) ([]EventSubmission, error) {
	rows, err := s.db.Query(s.rebind(`SELECT s.id, s.event_id, s.user_id, s.task_number, s.task_slug, s.task_title, s.query, s.passed, s.cases_json, s.submitted_at, COALESCE(p.username, '') FROM task_submissions s LEFT JOIN event_participants p ON p.event_id = s.event_id AND p.user_id = s.user_id WHERE s.event_id = ? ORDER BY s.submitted_at ASC, s.id ASC`), eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EventSubmission{}
	for rows.Next() {
		var item EventSubmission
		var casesJSON, submittedAt string
		if err := rows.Scan(&item.ID, &item.EventID, &item.UserID, &item.TaskNumber, &item.TaskSlug, &item.TaskTitle, &item.Query, &item.Passed, &casesJSON, &submittedAt, &item.Username); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(casesJSON), &item.Cases); err != nil {
			return nil, fmt.Errorf("invalid submission cases for id %d: %w", item.ID, err)
		}
		item.SubmittedAt, err = time.Parse(time.RFC3339Nano, submittedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
