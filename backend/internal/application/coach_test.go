package application

import (
	"context"
	"github.com/personalized-ai-coach/backend/internal/adapters/llm"
	"github.com/personalized-ai-coach/backend/internal/adapters/memory"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"strings"
	"testing"
	"time"
)

func testCoach(t *testing.T) (*Coach, *memory.Store, string) {
	t.Helper()
	s := memory.New()
	u, e := s.EnsureUser(context.Background(), "subject", "student@test")
	if e != nil {
		t.Fatal(e)
	}
	c := New(s, llm.Fake{})
	c.Now = func() time.Time { return time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC) }
	return c, s, u.ID
}
func TestDailyIsIdempotentAndPublishes(t *testing.T) {
	c, _, u := testCoach(t)
	a, created, e := c.Daily(context.Background(), u, "2026-07-26")
	if e != nil || !created {
		t.Fatalf("first: created=%v error=%v", created, e)
	}
	b, created, e := c.Daily(context.Background(), u, "2026-07-26")
	if e != nil || created || a.ID != b.ID {
		t.Fatalf("second not idempotent: %#v %#v %v", a, b, e)
	}
	if len(a.Quiz.Questions) != 3 || a.Status != "published" {
		t.Fatalf("incomplete session: %#v", a)
	}
}
func TestQuizSubmissionDoesNotDuplicateXP(t *testing.T) {
	c, s, u := testCoach(t)
	session, _, _ := c.Daily(context.Background(), u, "2026-07-26")
	var answers []domain.Answer
	for _, q := range session.Quiz.Questions {
		answers = append(answers, domain.Answer{QuestionID: q.ID, Value: q.CorrectAnswer, Confidence: 1})
	}
	r, e := c.SubmitQuiz(context.Background(), u, session.Quiz.ID, "same-key", answers)
	if e != nil {
		t.Fatal(e)
	}
	again, e := c.SubmitQuiz(context.Background(), u, session.Quiz.ID, "same-key", answers)
	if e != nil || again.AttemptID != r.AttemptID {
		t.Fatal("attempt was duplicated")
	}
	usr, _ := s.GetUser(context.Background(), u)
	if usr.XP != r.XPAwarded {
		t.Fatalf("xp=%d want %d", usr.XP, r.XPAwarded)
	}
}
func TestChunkTextOverlapAndIsolation(t *testing.T) {
	chunks := ChunkText("u", "d", strings.Repeat("word ", 20), 8, 2)
	if len(chunks) != 3 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	if chunks[0].UserID != "u" || chunks[0].Locator == "" {
		t.Fatalf("missing provenance: %#v", chunks[0])
	}
}
func TestInterviewReachesScorecard(t *testing.T) {
	c, _, u := testCoach(t)
	x, e := c.CreateInterview(context.Background(), u, "Design a ride sharing platform")
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 6; i++ {
		x, e = c.InterviewReply(context.Background(), u, x.ID, "Requirements scale cache partition failure retry auth encrypt cost trade-off assumption")
		if e != nil {
			t.Fatal(e)
		}
	}
	if x.State != "scored" || x.Scorecard == nil || x.Scorecard.Overall < 50 {
		t.Fatalf("not scored: %#v", x)
	}
}
