package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var eventSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type CreateEventInput struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	StartsAt    time.Time  `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	TaskNumbers []int      `json:"taskNumbers"`
}

type EventPage struct {
	Event       Event               `json:"event"`
	Participant *EventParticipant   `json:"participant"`
	Started     bool                `json:"started"`
	Tasks       []PublicTaskSummary `json:"tasks"`
}

type AdminEventDetail struct {
	Event
	TaskNumbers      []int              `json:"taskNumbers"`
	Participants     []EventParticipant `json:"participants"`
	Submissions      []EventSubmission  `json:"submissions"`
	ParticipantCount int                `json:"participantCount"`
	SubmitterCount   int                `json:"submitterCount"`
	SubmissionCount  int                `json:"submissionCount"`
}

func (s *TaskService) CreateEvent(input CreateEventInput) (Event, error) {
	input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return Event{}, fmt.Errorf("event title is required")
	}
	if !eventSlugPattern.MatchString(input.Slug) {
		return Event{}, fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}
	if input.StartsAt.IsZero() {
		return Event{}, fmt.Errorf("startsAt is required")
	}
	if input.EndsAt != nil && !input.EndsAt.After(input.StartsAt) {
		return Event{}, fmt.Errorf("endsAt must be after startsAt")
	}
	if len(input.TaskNumbers) == 0 {
		return Event{}, fmt.Errorf("at least one task is required")
	}
	seen := map[int]bool{}
	for _, number := range input.TaskNumbers {
		if seen[number] {
			return Event{}, fmt.Errorf("task %d is duplicated", number)
		}
		seen[number] = true
		if _, err := s.LoadTask(strconv.Itoa(number)); err != nil {
			return Event{}, err
		}
	}
	return s.submissions.CreateEvent(Event{
		Slug: input.Slug, Title: input.Title, StartsAt: input.StartsAt.UTC(), EndsAt: utcTimePointer(input.EndsAt), CreatedAt: time.Now().UTC(),
	}, input.TaskNumbers)
}

func utcTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func (s *TaskService) Events() ([]EventSummary, error) {
	return s.submissions.ListEvents()
}

func (s *TaskService) EventPage(slug, userID string) (EventPage, error) {
	event, err := s.submissions.EventBySlug(slug)
	if err != nil {
		return EventPage{}, fmt.Errorf("event %s was not found", slug)
	}
	participant, err := s.submissions.Participant(event.ID, userID)
	if err != nil {
		return EventPage{}, err
	}
	page := EventPage{Event: event, Participant: participant, Started: !time.Now().UTC().Before(event.StartsAt), Tasks: []PublicTaskSummary{}}
	if participant == nil || !page.Started {
		return page, nil
	}
	page.Tasks, err = s.eventTaskSummaries(event.ID, userID)
	return page, err
}

func (s *TaskService) JoinEvent(slug, userID, username string) (*EventParticipant, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len([]rune(username)) > 40 {
		return nil, fmt.Errorf("username must be 40 characters or fewer")
	}
	event, err := s.submissions.EventBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("event %s was not found", slug)
	}
	return s.submissions.JoinEvent(event.ID, userID, username)
}

func (s *TaskService) GetEventTask(slug, number, userID string) (PublicTask, error) {
	event, task, err := s.eventTaskAccess(slug, number, userID)
	if err != nil {
		return PublicTask{}, err
	}
	progress, err := s.submissions.ProgressByUserEvent(userID, event.ID)
	if err != nil {
		return PublicTask{}, err
	}
	result := publicTask(task)
	applyProgressToTask(&result, progress[task.Number])
	return result, nil
}

func (s *TaskService) SubmitEventTask(slug, number, query, userID string) (TaskSubmitResult, error) {
	event, _, err := s.eventTaskAccess(slug, number, userID)
	if err != nil {
		return TaskSubmitResult{}, err
	}
	return s.submit(number, query, userID, &event.ID)
}

func (s *TaskService) EventTaskSubmissions(slug, number, userID string) ([]TaskSubmission, error) {
	event, task, err := s.eventTaskAccess(slug, number, userID)
	if err != nil {
		return nil, err
	}
	return s.submissions.ListByUserTask(userID, task.Number, &event.ID)
}

func (s *TaskService) eventTaskAccess(slug, number, userID string) (Event, Task, error) {
	event, err := s.submissions.EventBySlug(slug)
	if err != nil {
		return Event{}, Task{}, fmt.Errorf("event %s was not found", slug)
	}
	participant, err := s.submissions.Participant(event.ID, userID)
	if err != nil {
		return Event{}, Task{}, err
	}
	if participant == nil {
		return Event{}, Task{}, fmt.Errorf("join the event before opening tasks")
	}
	if time.Now().UTC().Before(event.StartsAt) {
		return Event{}, Task{}, fmt.Errorf("the event has not started")
	}
	task, err := s.LoadTask(number)
	if err != nil {
		return Event{}, Task{}, err
	}
	allowed, err := s.submissions.EventHasTask(event.ID, task.Number)
	if err != nil {
		return Event{}, Task{}, err
	}
	if !allowed {
		return Event{}, Task{}, fmt.Errorf("task %d is not part of this event", task.Number)
	}
	return event, task, nil
}

func (s *TaskService) eventTaskSummaries(eventID int64, userID string) ([]PublicTaskSummary, error) {
	numbers, err := s.submissions.EventTaskNumbers(eventID)
	if err != nil {
		return nil, err
	}
	progress, err := s.submissions.ProgressByUserEvent(userID, eventID)
	if err != nil {
		return nil, err
	}
	items := make([]PublicTaskSummary, 0, len(numbers))
	for _, number := range numbers {
		task, err := s.LoadTask(strconv.Itoa(number))
		if err != nil {
			return nil, err
		}
		item := publicTaskSummary(task)
		applyProgressToSummary(&item, progress[number])
		items = append(items, item)
	}
	return items, nil
}

func (s *TaskService) AdminEvent(slug string) (AdminEventDetail, error) {
	event, err := s.submissions.EventBySlug(slug)
	if err != nil {
		return AdminEventDetail{}, fmt.Errorf("event %s was not found", slug)
	}
	numbers, err := s.submissions.EventTaskNumbers(event.ID)
	if err != nil {
		return AdminEventDetail{}, err
	}
	participants, err := s.submissions.EventParticipants(event.ID)
	if err != nil {
		return AdminEventDetail{}, err
	}
	submissions, err := s.submissions.EventSubmissions(event.ID)
	if err != nil {
		return AdminEventDetail{}, err
	}
	submitters := map[string]bool{}
	for index := range submissions {
		submissions[index].BenchmarkEnabled = s.taskBenchmarkEnabled(submissions[index].TaskNumber)
		submission := submissions[index]
		submitters[submission.UserID] = true
	}
	return AdminEventDetail{
		Event: event, TaskNumbers: numbers, Participants: participants, Submissions: submissions,
		ParticipantCount: len(participants), SubmitterCount: len(submitters), SubmissionCount: len(submissions),
	}, nil
}
