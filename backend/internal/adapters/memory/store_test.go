package memory

import (
	"context"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"sync"
	"testing"
	"time"
)

func TestSessionCompletionIsIdempotentAndTimezoneAware(t *testing.T) {
	ctx := context.Background()
	store := New()
	user, err := store.EnsureUser(ctx, "learner", "learner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	preferences := user.Preferences
	preferences.Timezone = "Asia/Bangkok"
	if _, err = store.SavePreferences(ctx, user.ID, preferences); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 7, 25, 16, 30, 0, 0, time.UTC) // 23:30 in Bangkok
	streak, created, err := store.RecordSessionCompletion(ctx, user.ID, "session-1", first, 45, 25)
	if err != nil || !created || streak != 1 {
		t.Fatalf("first completion = streak %d created %v err %v", streak, created, err)
	}
	streak, created, err = store.RecordSessionCompletion(ctx, user.ID, "session-1", first, 45, 25)
	if err != nil || created || streak != 1 {
		t.Fatalf("duplicate completion = streak %d created %v err %v", streak, created, err)
	}
	second := first.Add(24 * time.Hour)
	streak, created, err = store.RecordSessionCompletion(ctx, user.ID, "session-2", second, 30, 25)
	if err != nil || !created || streak != 2 {
		t.Fatalf("second-day completion = streak %d created %v err %v", streak, created, err)
	}
	days, err := store.Activity(ctx, user.ID)
	if err != nil || len(days) != 2 || days[0].StudyMinutes != 45 || days[1].StudyMinutes != 30 {
		t.Fatalf("activity = %#v err %v", days, err)
	}
}

func TestConcurrentDistinctQuizAttemptsDoNotLoseKnowledgeUpdates(t *testing.T) {
	ctx := context.Background()
	store := New()
	user, err := store.EnsureUser(ctx, "learner", "learner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	observation := ports.KnowledgeObservation{NodeID: "node", Topic: "System Design.Caching", Domain: "System Design", Quality: 5, Confidence: 1, At: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)}
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	errors := make(chan error, 2)
	for _, scope := range []string{"quiz:key-a", "quiz:key-b"} {
		scope := scope
		go func() {
			defer wait.Done()
			<-start
			_, _, commitErr := store.CommitQuizAttempt(ctx, user.ID, scope, scope, domain.QuizResult{AttemptID: scope, XPAwarded: 10}, []ports.KnowledgeObservation{observation})
			errors <- commitErr
		}()
	}
	close(start)
	wait.Wait()
	close(errors)
	for commitErr := range errors {
		if commitErr != nil {
			t.Fatal(commitErr)
		}
	}
	node, ok, err := store.KnowledgeNode(ctx, user.ID, observation.Topic)
	if err != nil || !ok {
		t.Fatalf("knowledge node missing: ok=%v err=%v", ok, err)
	}
	if node.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", node.Attempts)
	}
	savedUser, err := store.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if savedUser.XP != 20 {
		t.Fatalf("xp = %d, want 20", savedUser.XP)
	}
}

func TestCompareAndSwapInterviewRejectsStaleUpdate(t *testing.T) {
	ctx := context.Background()
	store := New()
	interview := domain.Interview{ID: "interview", UserID: "user", Sequence: 1, State: "created", Messages: []domain.InterviewMessage{{Sequence: 1, Role: "interviewer", Content: "start"}}}
	if err := store.SaveInterview(ctx, interview); err != nil {
		t.Fatal(err)
	}
	first := interview
	first.Sequence = 3
	first.Messages = append(first.Messages, domain.InterviewMessage{Sequence: 2, Role: "candidate", Content: "first"})
	if _, updated, err := store.CompareAndSwapInterview(ctx, "user", "interview", 1, first); err != nil || !updated {
		t.Fatalf("first update: updated=%v err=%v", updated, err)
	}
	stale := interview
	stale.Sequence = 3
	stale.Messages = append(stale.Messages, domain.InterviewMessage{Sequence: 2, Role: "candidate", Content: "stale"})
	saved, updated, err := store.CompareAndSwapInterview(ctx, "user", "interview", 1, stale)
	if err != nil || updated {
		t.Fatalf("stale update: updated=%v err=%v", updated, err)
	}
	if len(saved.Messages) != 2 || saved.Messages[1].Content != "first" {
		t.Fatalf("stale update overwrote interview: %#v", saved.Messages)
	}
}

func TestPublishEventRetainsLatestThousandPerUser(t *testing.T) {
	ctx := context.Background()
	store := New()
	for i := 0; i < maxEventsPerUser+5; i++ {
		if err := store.PublishEvent(ctx, "user", domain.Event{Type: "workflow.updated"}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := store.PublishEvent(ctx, "other-user", domain.Event{Type: "workflow.updated"}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.EventsSince(ctx, "user", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != maxEventsPerUser {
		t.Fatalf("events = %d, want %d", len(events), maxEventsPerUser)
	}
	if events[0].Sequence != 6 || events[len(events)-1].Sequence != maxEventsPerUser+5 {
		t.Fatalf("unexpected retained sequence range %d..%d", events[0].Sequence, events[len(events)-1].Sequence)
	}
	oldest, latest, err := store.EventBounds(ctx, "user")
	if err != nil || oldest != 6 || latest != maxEventsPerUser+5 {
		t.Fatalf("user bounds = %d..%d err=%v", oldest, latest, err)
	}
	otherOldest, otherLatest, err := store.EventBounds(ctx, "other-user")
	if err != nil || otherOldest != 1 || otherLatest != 2 {
		t.Fatalf("other-user bounds = %d..%d err=%v", otherOldest, otherLatest, err)
	}
	otherEvents, err := store.EventsSince(ctx, "other-user", 0)
	if err != nil || len(otherEvents) != 2 || otherEvents[0].Sequence != 1 || otherEvents[1].Sequence != 2 {
		t.Fatalf("other-user events = %#v err=%v", otherEvents, err)
	}
	emptyOldest, emptyLatest, err := store.EventBounds(ctx, "empty-user")
	if err != nil || emptyOldest != 0 || emptyLatest != 0 {
		t.Fatalf("empty-user bounds = %d..%d err=%v", emptyOldest, emptyLatest, err)
	}
}
