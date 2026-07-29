package ports

import (
	"context"
	"errors"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"io"
	"time"
)

var (
	ErrIdempotencyConflict       = errors.New("idempotency key was already used with different input")
	ErrInterviewSequenceConflict = errors.New("interview client sequence was already used")
)

type KnowledgeObservation struct {
	NodeID     string
	Topic      string
	Domain     string
	Quality    int
	Confidence float64
	At         time.Time
}

type Store interface {
	EnsureUser(context.Context, string, string) (domain.User, error)
	GetUser(context.Context, string) (domain.User, error)
	SavePreferences(context.Context, string, domain.Preferences) (domain.User, error)
	GetDailySession(context.Context, string, string) (domain.DailySession, bool, error)
	CreateDailySession(context.Context, domain.DailySession) (domain.DailySession, bool, error)
	SaveDailySession(context.Context, domain.DailySession) error
	GetSession(context.Context, string, string) (domain.DailySession, bool, error)
	SaveWorkflow(context.Context, domain.Workflow) error
	GetWorkflow(context.Context, string, string) (domain.Workflow, bool, error)
	PublishEvent(context.Context, string, domain.Event) error
	EventsSince(context.Context, string, int64) ([]domain.Event, error)
	EventBounds(context.Context, string) (int64, int64, error)
	GetQuiz(context.Context, string, string) (domain.Quiz, bool, error)
	SaveQuiz(context.Context, string, domain.Quiz) error
	SaveQuizResult(context.Context, string, string, domain.QuizResult) (domain.QuizResult, bool, error)
	CommitQuizAttempt(context.Context, string, string, string, domain.QuizResult, []KnowledgeObservation) (domain.QuizResult, bool, error)
	KnowledgeNode(context.Context, string, string) (domain.KnowledgeNode, bool, error)
	SaveKnowledgeNode(context.Context, domain.KnowledgeNode) error
	AddXP(context.Context, string, string, int) (int, bool, error)
	RecordSessionCompletion(context.Context, string, string, time.Time, int, int) (int, bool, error)
	Activity(context.Context, string) ([]domain.ActivityDay, error)
	SaveDocument(context.Context, domain.Document, []domain.Chunk) error
	Documents(context.Context, string) ([]domain.Document, error)
	Document(context.Context, string, string) (domain.Document, bool, error)
	DeleteDocument(context.Context, string, string) error
	SearchChunks(context.Context, string, string, int) ([]domain.Chunk, error)
	SaveInterview(context.Context, domain.Interview) error
	Interview(context.Context, string, string) (domain.Interview, bool, error)
	CompareAndSwapInterview(context.Context, string, string, int64, domain.Interview) (domain.Interview, bool, error)
	KnowledgeGraph(context.Context, string) ([]domain.KnowledgeNode, error)
}

type LLMRequest struct {
	Task, System, Prompt, Model string
	JSON                        bool
}
type LLMResponse struct {
	Content                   string
	InputTokens, OutputTokens int
	Model                     string
}
type LLM interface {
	Complete(context.Context, LLMRequest) (LLMResponse, error)
}
type ObjectStore interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Delete(context.Context, string) error
}
type Clock interface{ Now() time.Time }
