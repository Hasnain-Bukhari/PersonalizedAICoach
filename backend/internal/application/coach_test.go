package application

import (
	"context"
	"errors"
	"github.com/personalized-ai-coach/backend/internal/adapters/llm"
	"github.com/personalized-ai-coach/backend/internal/adapters/memory"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

type llmFunc func(context.Context, ports.LLMRequest) (ports.LLMResponse, error)

func (f llmFunc) Complete(ctx context.Context, request ports.LLMRequest) (ports.LLMResponse, error) {
	return f(ctx, request)
}

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

func TestDailyConcurrentCreationReturnsOneSession(t *testing.T) {
	c, _, u := testCoach(t)
	const callers = 16
	start := make(chan struct{})
	type outcome struct {
		session domain.DailySession
		created bool
		err     error
	}
	outcomes := make(chan outcome, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			session, created, err := c.Daily(context.Background(), u, "2026-07-26")
			outcomes <- outcome{session: session, created: created, err: err}
		}()
	}
	close(start)
	var sessionID string
	createdCount := 0
	for i := 0; i < callers; i++ {
		outcome := <-outcomes
		if outcome.err != nil {
			t.Fatal(outcome.err)
		}
		if outcome.created {
			createdCount++
		}
		if sessionID == "" {
			sessionID = outcome.session.ID
		}
		if outcome.session.ID != sessionID {
			t.Fatalf("got competing sessions %q and %q", sessionID, outcome.session.ID)
		}
	}
	if createdCount != 1 {
		t.Fatalf("created count = %d, want 1", createdCount)
	}
}

func TestUploadedTextInfluencesDailySessionWithTruthfulCitationTitle(t *testing.T) {
	c, _, userID := testCoach(t)
	document, err := c.Upload(
		context.Background(),
		userID,
		"system-design-notes.md",
		"text/markdown",
		int64(len("System Design.Fundamentals requires explicit failure budgets.")),
		strings.NewReader("System Design.Fundamentals requires explicit failure budgets."),
	)
	if err != nil {
		t.Fatal(err)
	}
	session, _, err := c.Daily(context.Background(), userID, "2026-07-27")
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Lesson.Citations) != 1 {
		t.Fatalf("citations = %#v, want uploaded document citation", session.Lesson.Citations)
	}
	citation := session.Lesson.Citations[0]
	if citation.DocumentID != document.ID || citation.Title != document.Name {
		t.Fatalf("citation provenance = %#v, want document %q titled %q", citation, document.ID, document.Name)
	}
	if !strings.Contains(citation.Quote, "failure budgets") {
		t.Fatalf("citation quote %q does not contain uploaded content", citation.Quote)
	}
}

func TestQuizSubmissionDoesNotDuplicateXP(t *testing.T) {
	c, s, u := testCoach(t)
	session, _, _ := c.Daily(context.Background(), u, "2026-07-26")
	answers := correctQuizAnswers(session.Quiz)
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

func TestConcurrentQuizRetryIsAtomicAndIdempotent(t *testing.T) {
	c, s, u := testCoach(t)
	session, _, err := c.Daily(context.Background(), u, "2026-07-26")
	if err != nil {
		t.Fatal(err)
	}
	answers := correctQuizAnswers(session.Quiz)
	const callers = 20
	start := make(chan struct{})
	results := make(chan domain.QuizResult, callers)
	errors := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			result, submitErr := c.SubmitQuiz(context.Background(), u, session.Quiz.ID, "retry-key", answers)
			results <- result
			errors <- submitErr
		}()
	}
	close(start)
	var attemptID string
	for i := 0; i < callers; i++ {
		if submitErr := <-errors; submitErr != nil {
			t.Fatal(submitErr)
		}
		result := <-results
		if attemptID == "" {
			attemptID = result.AttemptID
		}
		if result.AttemptID != attemptID {
			t.Fatalf("duplicate attempt %q, want %q", result.AttemptID, attemptID)
		}
	}
	user, err := s.GetUser(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if user.XP != 25 {
		t.Fatalf("xp = %d, want one 25-point award", user.XP)
	}
	node, ok, err := s.KnowledgeNode(context.Background(), u, "System Design.Fundamentals")
	if err != nil || !ok {
		t.Fatalf("knowledge node missing: ok=%v err=%v", ok, err)
	}
	if node.Attempts != len(session.Quiz.Questions) {
		t.Fatalf("attempts = %d, want %d", node.Attempts, len(session.Quiz.Questions))
	}
	if node.Confidence != 100 {
		t.Fatalf("confidence = %v, want 100", node.Confidence)
	}
}

func TestQuizIdempotencyKeyRejectsDifferentAnswers(t *testing.T) {
	c, _, u := testCoach(t)
	session, _, err := c.Daily(context.Background(), u, "2026-07-26")
	if err != nil {
		t.Fatal(err)
	}
	first := correctQuizAnswers(session.Quiz)
	if _, err = c.SubmitQuiz(context.Background(), u, session.Quiz.ID, "same-key", first); err != nil {
		t.Fatal(err)
	}
	first[0].Value = "different answer"
	if _, err = c.SubmitQuiz(context.Background(), u, session.Quiz.ID, "same-key", first); !errors.Is(err, ports.ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
}

func correctQuizAnswers(quiz domain.Quiz) []domain.Answer {
	answers := make([]domain.Answer, 0, len(quiz.Questions))
	for _, question := range quiz.Questions {
		answers = append(answers, domain.Answer{QuestionID: question.ID, Value: question.CorrectAnswer, Confidence: 1})
	}
	return answers
}

func TestSubmitQuizRejectsInvalidDirectCallsWithoutMutation(t *testing.T) {
	tests := map[string]func(domain.Quiz) []domain.Answer{
		"empty answers": func(domain.Quiz) []domain.Answer { return nil },
		"missing question": func(quiz domain.Quiz) []domain.Answer {
			return correctQuizAnswers(quiz)[:len(quiz.Questions)-1]
		},
		"duplicate question": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[len(answers)-1].QuestionID = answers[0].QuestionID
			return answers
		},
		"unknown question": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[0].QuestionID = "unknown-question"
			return answers
		},
		"empty value": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[0].Value = "  "
			return answers
		},
		"negative confidence": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[0].Confidence = -0.01
			return answers
		},
		"confidence above one": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[0].Confidence = 1.01
			return answers
		},
		"non-finite confidence": func(quiz domain.Quiz) []domain.Answer {
			answers := correctQuizAnswers(quiz)
			answers[0].Confidence = math.NaN()
			return answers
		},
	}
	for name, invalidAnswers := range tests {
		t.Run(name, func(t *testing.T) {
			coach, store, userID := testCoach(t)
			session, _, err := coach.Daily(context.Background(), userID, "2026-07-26")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = coach.SubmitQuiz(context.Background(), userID, session.Quiz.ID, "invalid-key", invalidAnswers(session.Quiz)); err == nil {
				t.Fatal("invalid quiz submission succeeded")
			}
			user, err := store.GetUser(context.Background(), userID)
			if err != nil {
				t.Fatal(err)
			}
			if user.XP != 0 {
				t.Fatalf("XP mutated to %d", user.XP)
			}
			nodes, err := store.KnowledgeGraph(context.Background(), userID)
			if err != nil {
				t.Fatal(err)
			}
			if len(nodes) != 0 {
				t.Fatalf("mastery mutated: %#v", nodes)
			}
		})
	}
}

func TestGeneratedMaterialRejectsIncompleteOrInvalidContent(t *testing.T) {
	invalid := `{"objectives":["Learn"],"simple":"simple","real_world":"real","advanced":"advanced","diagram":"graph LR; A-->B","best_practices":"best","pitfalls":"pitfalls","cheat_sheet":"cheat","confidence":1.2,"questions":[{"type":"multiple_choice","prompt":"p","options":["a","b"],"answer":"a","explanation":"e"},{"type":"scenario","prompt":"p","answer":"a","explanation":"e"},{"type":"true_false","prompt":"p","options":["true","false"],"answer":"true","explanation":"e"}]}`
	c := New(memory.New(), llmFunc(func(context.Context, ports.LLMRequest) (ports.LLMResponse, error) {
		return ports.LLMResponse{Content: invalid}, nil
	}))
	lesson, quiz := c.generateLearningMaterial(context.Background(), "topic", nil)
	if lesson.Confidence != .82 || len(quiz.Questions) != 3 || quiz.Questions[0].Prompt != "Which step should come before selecting infrastructure?" {
		t.Fatalf("invalid generated material was accepted: lesson=%#v quiz=%#v", lesson, quiz)
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

type interviewBarrierStore struct {
	ports.Store
	ready sync.WaitGroup
	mu    sync.Mutex
	calls int
}

func newInterviewBarrierStore(store ports.Store) *interviewBarrierStore {
	barrier := &interviewBarrierStore{Store: store}
	barrier.ready.Add(2)
	return barrier
}

func (s *interviewBarrierStore) Interview(ctx context.Context, user, id string) (domain.Interview, bool, error) {
	interview, ok, err := s.Store.Interview(ctx, user, id)
	s.mu.Lock()
	wait := s.calls < 2
	if wait {
		s.calls++
	}
	s.mu.Unlock()
	if wait {
		s.ready.Done()
		s.ready.Wait()
	}
	return interview, ok, err
}

func TestInterviewEventIsDeduplicatedAcrossConcurrentDeliveryAndReconnect(t *testing.T) {
	base := memory.New()
	user, err := base.EnsureUser(context.Background(), "subject", "student@test")
	if err != nil {
		t.Fatal(err)
	}
	creator := New(base, llm.Fake{})
	interview, err := creator.CreateInterview(context.Background(), user.ID, "Design a queue")
	if err != nil {
		t.Fatal(err)
	}
	barrier := newInterviewBarrierStore(base)
	coach := New(barrier, llm.Fake{})
	results := make(chan domain.Interview, 2)
	created := make(chan bool, 2)
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			result, applied, replyErr := coach.InterviewReplyEvent(context.Background(), user.ID, interview.ID, "event-1", 1, "same response")
			results <- result
			created <- applied
			errors <- replyErr
		}()
	}
	appliedCount := 0
	for i := 0; i < 2; i++ {
		if replyErr := <-errors; replyErr != nil {
			t.Fatal(replyErr)
		}
		<-results
		if <-created {
			appliedCount++
		}
	}
	if appliedCount != 1 {
		t.Fatalf("applied count = %d, want 1", appliedCount)
	}
	reconnected := New(base, llm.Fake{})
	replayed, applied, err := reconnected.InterviewReplyEvent(context.Background(), user.ID, interview.ID, "event-1", 1, "same response")
	if err != nil || applied {
		t.Fatalf("reconnected retry: applied=%v err=%v", applied, err)
	}
	if replayed.Sequence != 3 {
		t.Fatalf("reconnected retry advanced to sequence %d", replayed.Sequence)
	}
	stale, applied, err := reconnected.InterviewReplyEvent(context.Background(), user.ID, interview.ID, "event-stale", 1, "different response")
	if !errors.Is(err, ports.ErrInterviewSequenceConflict) || applied || stale.Sequence != 3 {
		t.Fatalf("stale client sequence: sequence=%d applied=%v err=%v", stale.Sequence, applied, err)
	}
	saved, ok, err := base.Interview(context.Background(), user.ID, interview.ID)
	if err != nil || !ok {
		t.Fatalf("saved interview missing: ok=%v err=%v", ok, err)
	}
	candidates := 0
	for _, message := range saved.Messages {
		if message.Role == "candidate" {
			candidates++
		}
	}
	if candidates != 1 || saved.Sequence != 3 {
		t.Fatalf("interview was not deduplicated: sequence=%d messages=%#v", saved.Sequence, saved.Messages)
	}
}

func TestDistinctInterviewEventSequenceConflictCanRetryAtNextSequence(t *testing.T) {
	c, store, userID := testCoach(t)
	interview, err := c.CreateInterview(context.Background(), userID, "Design a queue")
	if err != nil {
		t.Fatal(err)
	}
	winner, applied, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-winner", 1, "first response")
	if err != nil || !applied || winner.Sequence != 3 {
		t.Fatalf("winner: sequence=%d applied=%v err=%v", winner.Sequence, applied, err)
	}
	loser, applied, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-loser", 1, "second response")
	if !errors.Is(err, ports.ErrInterviewSequenceConflict) || applied || loser.Sequence != winner.Sequence {
		t.Fatalf("loser: sequence=%d applied=%v err=%v", loser.Sequence, applied, err)
	}
	saved, ok, err := store.Interview(context.Background(), userID, interview.ID)
	if err != nil || !ok {
		t.Fatalf("saved interview missing: ok=%v err=%v", ok, err)
	}
	if saved.Sequence != 3 || saved.LastClientSequence != 1 || len(saved.ProcessedEvents) != 1 || len(saved.Messages) != 3 {
		t.Fatalf("sequence conflict mutated interview: %#v", saved)
	}
	retried, applied, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-loser", 2, "second response")
	if err != nil || !applied || retried.Sequence != 5 || retried.LastClientSequence != 2 {
		t.Fatalf("retry: sequence=%d client_sequence=%d applied=%v err=%v", retried.Sequence, retried.LastClientSequence, applied, err)
	}
	exactRetry, applied, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-loser", 2, "second response")
	if err != nil || applied || exactRetry.Sequence != retried.Sequence {
		t.Fatalf("exact retry: sequence=%d applied=%v err=%v", exactRetry.Sequence, applied, err)
	}
	candidates := 0
	for _, message := range exactRetry.Messages {
		if message.Role == "candidate" {
			candidates++
		}
	}
	if candidates != 2 || len(exactRetry.ProcessedEvents) != 2 {
		t.Fatalf("retry did not apply exactly once: %#v", exactRetry)
	}
}

func TestInterviewEventIDRejectsChangedPayload(t *testing.T) {
	c, _, userID := testCoach(t)
	interview, err := c.CreateInterview(context.Background(), userID, "Design a queue")
	if err != nil {
		t.Fatal(err)
	}
	if _, applied, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-1", 1, "first response"); err != nil || !applied {
		t.Fatalf("first delivery: applied=%v err=%v", applied, err)
	}
	if _, _, err := c.InterviewReplyEvent(context.Background(), userID, interview.ID, "event-1", 1, "changed response"); !errors.Is(err, ports.ErrIdempotencyConflict) {
		t.Fatalf("changed event payload error = %v, want idempotency conflict", err)
	}
}
