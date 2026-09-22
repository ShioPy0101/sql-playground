package service

import (
	"reflect"
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
	resultB, err := service.SubmitEventTask(eventB.Slug, "1", "SELECT 3;", "user_same")
	if err != nil {
		t.Fatalf("event B submit returned error: %v", err)
	}
	if err := service.SaveBenchmarkReport(resultB.SubmissionID, "user_same", 1, BenchmarkReport{Status: "completed", Results: []BenchmarkResult{{RowCount: 1_000, VMSteps: 7}}}); err != nil {
		t.Fatalf("save event benchmark returned error: %v", err)
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
	if detail.Submissions[0].BenchmarkReport == nil || detail.Submissions[0].BenchmarkReport.Results[0].VMSteps != 7 {
		t.Fatalf("event B benchmark = %#v", detail.Submissions[0].BenchmarkReport)
	}
}

func TestUpdateEventTasksAddsRemovesReordersAndPreservesSubmissions(t *testing.T) {
	service := newTestTaskService(t)
	event, err := service.CreateEvent(CreateEventInput{Slug: "editable", Title: "Editable", StartsAt: time.Now().Add(-time.Hour), TaskNumbers: []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.JoinEvent(event.Slug, "user", "User"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitEventTask(event.Slug, "1", "SELECT 1;", "user"); err != nil {
		t.Fatal(err)
	}

	detail, err := service.UpdateEventTasks(event.Slug, []int{3, 2})
	if err != nil {
		t.Fatalf("UpdateEventTasks returned error: %v", err)
	}
	if !reflect.DeepEqual(detail.TaskNumbers, []int{3, 2}) {
		t.Fatalf("task numbers = %v", detail.TaskNumbers)
	}
	if len(detail.Submissions) != 1 || detail.Submissions[0].TaskNumber != 1 {
		t.Fatalf("submissions were changed: %#v", detail.Submissions)
	}
	page, err := service.EventPage(event.Slug, "user")
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{page.Tasks[0].Number, page.Tasks[1].Number}; !reflect.DeepEqual(got, []int{3, 2}) {
		t.Fatalf("participant order = %v", got)
	}

	rows, err := service.submissions.db.Query(`SELECT position FROM event_tasks WHERE event_id = ? ORDER BY position`, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	positions := []int{}
	for rows.Next() {
		var position int
		if err := rows.Scan(&position); err != nil {
			t.Fatal(err)
		}
		positions = append(positions, position)
	}
	if !reflect.DeepEqual(positions, []int{1, 2}) {
		t.Fatalf("positions = %v", positions)
	}
}

func TestUpdateEventTasksRejectsDuplicateUnknownTaskAndUnknownEvent(t *testing.T) {
	service := newTestTaskService(t)
	event, err := service.CreateEvent(CreateEventInput{Slug: "validation", Title: "Validation", StartsAt: time.Now(), TaskNumbers: []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	for name, testCase := range map[string]struct {
		slug    string
		numbers []int
	}{
		"duplicate":     {event.Slug, []int{1, 1}},
		"unknown task":  {event.Slug, []int{1, 999999}},
		"unknown event": {"missing", []int{1}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.UpdateEventTasks(testCase.slug, testCase.numbers); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	detail, err := service.AdminEvent(event.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(detail.TaskNumbers, []int{1, 2}) {
		t.Fatalf("tasks changed after validation error: %v", detail.TaskNumbers)
	}
}

func TestUpdateEventTasksRollsBackOnInsertFailure(t *testing.T) {
	service := newTestTaskService(t)
	event, err := service.CreateEvent(CreateEventInput{Slug: "rollback", Title: "Rollback", StartsAt: time.Now(), TaskNumbers: []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.submissions.UpdateEventTasks(event.ID, []int{3, 3}); err == nil {
		t.Fatal("expected duplicate insert error")
	}
	numbers, err := service.submissions.EventTaskNumbers(event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(numbers, []int{1, 2}) {
		t.Fatalf("rollback failed, tasks = %v", numbers)
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
