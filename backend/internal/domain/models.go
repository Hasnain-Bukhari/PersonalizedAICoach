package domain

import "time"

type User struct {
	ID            string      `json:"id"`
	Subject       string      `json:"-"`
	Email         string      `json:"email"`
	Preferences   Preferences `json:"preferences"`
	XP            int         `json:"xp"`
	CurrentStreak int         `json:"current_streak"`
}

type Preferences struct {
	Mode               string   `json:"mode"`
	Timezone           string   `json:"timezone"`
	SessionMinutes     int      `json:"session_minutes"`
	DailyTime          string   `json:"daily_time"`
	Domains            []string `json:"domains"`
	EmailNotifications bool     `json:"email_notifications"`
}

type KnowledgeNode struct {
	ID               string    `json:"id"`
	UserID           string    `json:"-"`
	Domain           string    `json:"domain"`
	TopicPath        string    `json:"topic_path"`
	Mastery          float64   `json:"mastery"`
	Confidence       float64   `json:"confidence"`
	EaseFactor       float64   `json:"ease_factor"`
	Repetitions      int       `json:"repetitions"`
	Attempts         int       `json:"attempts"`
	LastIntervalDays int       `json:"last_interval_days"`
	Mistakes         []string  `json:"mistakes"`
	LastStudied      time.Time `json:"last_studied"`
	NextRevisionDue  time.Time `json:"next_revision_due"`
}

type Citation struct {
	DocumentID string `json:"document_id"`
	ChunkID    string `json:"chunk_id"`
	Title      string `json:"title"`
	Locator    string `json:"locator"`
	Quote      string `json:"quote,omitempty"`
}

type Lesson struct {
	ID            string     `json:"id"`
	Topic         string     `json:"topic"`
	Objectives    []string   `json:"objectives"`
	Simple        string     `json:"simple"`
	RealWorld     string     `json:"real_world"`
	Advanced      string     `json:"advanced"`
	Diagram       string     `json:"diagram"`
	BestPractices string     `json:"best_practices"`
	Pitfalls      string     `json:"pitfalls"`
	CheatSheet    string     `json:"cheat_sheet"`
	Confidence    float64    `json:"confidence"`
	Citations     []Citation `json:"citations"`
}

type Question struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"-"`
	Explanation   string   `json:"-"`
	Topic         string   `json:"topic"`
}

type Quiz struct {
	ID        string     `json:"id"`
	Questions []Question `json:"questions"`
}

type DailySession struct {
	ID               string    `json:"id"`
	UserID           string    `json:"-"`
	Date             string    `json:"date"`
	Status           string    `json:"status"`
	WorkflowID       string    `json:"workflow_id"`
	Objectives       []string  `json:"objectives"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	Lesson           Lesson    `json:"lesson"`
	Quiz             Quiz      `json:"quiz"`
	Reflection       string    `json:"reflection"`
	Homework         string    `json:"homework"`
	Preview          string    `json:"preview"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Answer struct {
	QuestionID string  `json:"question_id"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}
type AnswerResult struct {
	QuestionID     string   `json:"question_id"`
	Correct        bool     `json:"correct"`
	Explanation    string   `json:"explanation"`
	Misconceptions []string `json:"misconceptions"`
}
type MasteryChange struct {
	Topic           string    `json:"topic"`
	Before          float64   `json:"before"`
	After           float64   `json:"after"`
	NextRevisionDue time.Time `json:"next_revision_due"`
}
type QuizResult struct {
	AttemptID      string          `json:"attempt_id"`
	Score          float64         `json:"score"`
	Results        []AnswerResult  `json:"results"`
	MasteryChanges []MasteryChange `json:"mastery_changes"`
	XPAwarded      int             `json:"xp_awarded"`
}

type Workflow struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Kind      string    `json:"kind"`
	State     string    `json:"state"`
	Error     string    `json:"error,omitempty"`
	Sequence  int64     `json:"sequence"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Event struct {
	ID         string    `json:"event_id"`
	Type       string    `json:"type"`
	WorkflowID string    `json:"workflow_id"`
	Sequence   int64     `json:"sequence"`
	Timestamp  time.Time `json:"timestamp"`
	Payload    any       `json:"payload"`
}

type Document struct {
	ID          string    `json:"id"`
	UserID      string    `json:"-"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Status      string    `json:"status"`
	Checksum    string    `json:"checksum"`
	Error       string    `json:"error,omitempty"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}
type Chunk struct {
	ID             string    `json:"id"`
	DocumentID     string    `json:"document_id"`
	UserID         string    `json:"-"`
	Text           string    `json:"text"`
	Heading        string    `json:"heading,omitempty"`
	Locator        string    `json:"locator"`
	EmbeddingModel string    `json:"embedding_model"`
	Sequence       int       `json:"sequence"`
	Embedding      []float32 `json:"-"`
}

type Interview struct {
	ID                 string                           `json:"id"`
	UserID             string                           `json:"-"`
	Prompt             string                           `json:"prompt"`
	State              string                           `json:"state"`
	Sequence           int64                            `json:"sequence"`
	LastClientSequence int64                            `json:"-"`
	ProcessedEvents    map[string]InterviewEventReceipt `json:"-"`
	Messages           []InterviewMessage               `json:"messages"`
	Scorecard          *Scorecard                       `json:"scorecard,omitempty"`
	CreatedAt          time.Time                        `json:"created_at"`
}
type InterviewEventReceipt struct {
	ClientSequence int64
	Fingerprint    string
}
type InterviewMessage struct {
	Sequence int64     `json:"sequence"`
	Role     string    `json:"role"`
	Content  string    `json:"content"`
	At       time.Time `json:"at"`
}
type Scorecard struct {
	Scalability   float64  `json:"scalability"`
	Reliability   float64  `json:"reliability"`
	Security      float64  `json:"security"`
	Cost          float64  `json:"cost"`
	Communication float64  `json:"communication"`
	Overall       float64  `json:"overall"`
	Strengths     []string `json:"strengths"`
	Improvements  []string `json:"improvements"`
}

type AgentRun struct {
	ID            string        `json:"id"`
	UserID        string        `json:"user_id"`
	WorkflowID    string        `json:"workflow_id"`
	AgentType     string        `json:"agent_type"`
	AgentVersion  string        `json:"agent_version"`
	PromptVersion string        `json:"prompt_version"`
	Model         string        `json:"model"`
	Status        string        `json:"status"`
	Input         []byte        `json:"input"`
	Output        []byte        `json:"output"`
	Citations     []Citation    `json:"citations"`
	InputTokens   int           `json:"input_tokens"`
	OutputTokens  int           `json:"output_tokens"`
	RetryCount    int           `json:"retry_count"`
	Latency       time.Duration `json:"latency"`
	CreatedAt     time.Time     `json:"created_at"`
}

type ActivityDay struct {
	Date         string `json:"date"`
	StudyMinutes int    `json:"study_minutes"`
	XP           int    `json:"xp"`
}
