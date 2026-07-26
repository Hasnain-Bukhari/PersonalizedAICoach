package domain

import (
	"testing"
	"time"
)

func TestScheduleSM2Boundaries(t *testing.T) {
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	cases := []struct{ q, want int }{{0, 1}, {2, 1}, {3, 7}, {4, 7}, {5, 14}}
	for _, tc := range cases {
		n := ScheduleSM2(KnowledgeNode{EaseFactor: 2.5}, tc.q, now)
		if n.LastIntervalDays != tc.want {
			t.Errorf("q=%d interval=%d want %d", tc.q, n.LastIntervalDays, tc.want)
		}
		if n.NextRevisionDue != now.AddDate(0, 0, tc.want) {
			t.Errorf("q=%d due=%v", tc.q, n.NextRevisionDue)
		}
	}
}
func TestScheduleSM2EaseFloorAndGrowth(t *testing.T) {
	now := time.Now()
	n := ScheduleSM2(KnowledgeNode{EaseFactor: 1.3, LastIntervalDays: 20, Repetitions: 2}, 0, now)
	if n.EaseFactor != 1.3 {
		t.Fatalf("ease floor violated: %v", n.EaseFactor)
	}
	n = ScheduleSM2(KnowledgeNode{EaseFactor: 2.5, LastIntervalDays: 20, Repetitions: 2}, 5, now)
	if n.LastIntervalDays != 52 {
		t.Fatalf("perfect interval=%d want 52", n.LastIntervalDays)
	}
}
func TestMasteryIsBoundedAndWeighted(t *testing.T) {
	if got := UpdateMastery(50, 100, 4); got != 67.5 {
		t.Fatalf("got %v", got)
	}
	if got := UpdateMastery(99, 200, 4); got > 100 {
		t.Fatalf("not bounded: %v", got)
	}
}
