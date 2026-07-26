package memory

import (
	"context"
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
