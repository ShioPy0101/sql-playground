package service

import (
	"testing"
	"time"
)

func TestEventSubmissionsAreIsolatedFromNormalAndOtherEvents(t *testing.T) {
	service := newTestTaskService(t)
	startsAt := time.Now().Add(-time.Hour)

	eventA, err := service.CreateEvent(CreateEventInput{Slug: "event-a", Title: "Event A", StartsAt: startsAt, TaskNumbers: []int{1}})
	if err != nil {
		t.Fatalf("CreateEvent A returned error: %v", err)
	}
	eventB, err := service.CreateEvent(CreateEventInput{Slug: "event-b", Title: "Event B", StartsAt: startsAt, TaskNumbers: []int{1}})
	if err != nil {
		t.Fatalf("CreateEvent B returned error: %v", err)
	}

	if _, err := service.JoinEvent(eventA.Slug, "user_same", "Alice"); err != nil {
		t.Fatalf("JoinEvent A returned error: %v", err)
	}
	if _, err := service.JoinEvent(eventB.Slug, "user_same", "Alice B"); err != nil {
		t.Fatalf("JoinEvent B returned error: %v", err)
	}
	if _, err := service.SubmitForUser("1", "SELECT 1;", "user_same"); err != nil {
		t.Fatalf("normal submit returned error: %v", err)
	}
	if _, err := service.SubmitEventTask(eventA.Slug, "1", "SELECT 2;", "user_same"); err != nil {
		t.Fatalf("event A submit returned error: %v", err)
	}
	if _, err := service.SubmitEventTask(eventB.Slug, "1", "SELECT 3;", "user_same"); err != nil {
		t.Fatalf("event B submit returned error: %v", err)
	}

	normal, err := service.SubmissionsForUserTask("1", "user_same")
	if err != nil {
		t.Fatalf("normal history returned error: %v", err)
	}
	if len(normal) != 1 || normal[0].EventID != nil || normal[0].Query != "SELECT 1;" {
		t.Fatalf("normal history = %#v", normal)
	}

	historyA, err := service.EventTaskSubmissions(eventA.Slug, "1", "user_same")
	if err != nil {
		t.Fatalf("event A history returned error: %v", err)
	}
	if len(historyA) != 1 || historyA[0].EventID == nil || *historyA[0].EventID != eventA.ID || historyA[0].Query != "SELECT 2;" {
		t.Fatalf("event A history = %#v", historyA)
	}

	detail, err := service.AdminEvent(eventB.Slug)
	if err != nil {
		t.Fatalf("AdminEvent returned error: %v", err)
	}
	if detail.SubmissionCount != 1 || detail.Submissions[0].Username != "Alice B" || detail.Submissions[0].Query != "SELECT 3;" {
		t.Fatalf("event B detail = %#v", detail)
	}
}

func TestEventCannotExposeTasksBeforeStart(t *testing.T) {
	service := newTestTaskService(t)
	event, err := service.CreateEvent(CreateEventInput{Slug: "future", Title: "Future", StartsAt: time.Now().Add(time.Hour), TaskNumbers: []int{1}})
	if err != nil {
		t.Fatalf("CreateEvent returned error: %v", err)
	}
	if _, err := service.JoinEvent(event.Slug, "user_waiting", "Waiting"); err != nil {
		t.Fatalf("JoinEvent returned error: %v", err)
	}

	page, err := service.EventPage(event.Slug, "user_waiting")
	if err != nil {
		t.Fatalf("EventPage returned error: %v", err)
	}
	if page.Started || len(page.Tasks) != 0 {
		t.Fatalf("page before start = %#v", page)
	}
	if _, err := service.GetEventTask(event.Slug, "1", "user_waiting"); err == nil {
		t.Fatalf("GetEventTask before start returned nil error")
	}
}
