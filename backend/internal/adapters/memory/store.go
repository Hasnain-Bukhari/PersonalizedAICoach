package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrConflict = ports.ErrIdempotencyConflict

const maxEventsPerUser = 1000

type Store struct {
	mu             sync.RWMutex
	users          map[string]domain.User
	sessions       map[string]domain.DailySession
	workflows      map[string]domain.Workflow
	events         map[string][]domain.Event
	quizzes        map[string]domain.Quiz
	results        map[string]domain.QuizResult
	resultInputs   map[string]string
	nodes          map[string]domain.KnowledgeNode
	xpKeys         map[string]bool
	completionKeys map[string]bool
	activity       map[string]map[string]domain.ActivityDay
	documents      map[string]domain.Document
	chunks         []domain.Chunk
	interviews     map[string]domain.Interview
}

func New() *Store {
	return &Store{users: map[string]domain.User{}, sessions: map[string]domain.DailySession{}, workflows: map[string]domain.Workflow{}, events: map[string][]domain.Event{}, quizzes: map[string]domain.Quiz{}, results: map[string]domain.QuizResult{}, resultInputs: map[string]string{}, nodes: map[string]domain.KnowledgeNode{}, xpKeys: map[string]bool{}, completionKeys: map[string]bool{}, activity: map[string]map[string]domain.ActivityDay{}, documents: map[string]domain.Document{}, interviews: map[string]domain.Interview{}}
}
func key(a, b string) string { return a + ":" + b }
func (s *Store) EnsureUser(_ context.Context, subject, email string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[subject]; ok {
		return u, nil
	}
	u := domain.User{ID: newUUID(), Subject: subject, Email: email, Preferences: domain.Preferences{Mode: "Teacher", Timezone: "UTC", SessionMinutes: 45, DailyTime: "20:00"}}
	s.users[subject] = u
	s.users[u.ID] = u
	return u, nil
}
func (s *Store) GetUser(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return u, errors.New("user not found")
	}
	return u, nil
}
func (s *Store) SavePreferences(_ context.Context, id string, p domain.Preferences) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return u, errors.New("user not found")
	}
	u.Preferences = p
	s.users[id] = u
	s.users[u.Subject] = u
	return u, nil
}
func (s *Store) GetDailySession(_ context.Context, u, d string) (domain.DailySession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.sessions[key(u, d)]
	return x, ok, nil
}
func (s *Store) CreateDailySession(_ context.Context, x domain.DailySession) (domain.DailySession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionKey := key(x.UserID, x.Date)
	if saved, ok := s.sessions[sessionKey]; ok {
		return saved, false, nil
	}
	s.sessions[sessionKey] = x
	s.sessions[key(x.UserID, x.ID)] = x
	s.quizzes[key(x.UserID, x.Quiz.ID)] = x.Quiz
	return x, true, nil
}
func (s *Store) SaveDailySession(_ context.Context, x domain.DailySession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[key(x.UserID, x.Date)] = x
	s.sessions[key(x.UserID, x.ID)] = x
	return nil
}
func (s *Store) GetSession(_ context.Context, u, id string) (domain.DailySession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.sessions[key(u, id)]
	return x, ok, nil
}
func (s *Store) SaveWorkflow(_ context.Context, x domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[key(x.UserID, x.ID)] = x
	return nil
}
func (s *Store) GetWorkflow(_ context.Context, u, id string) (domain.Workflow, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.workflows[key(u, id)]
	return x, ok, nil
}
func (s *Store) PublishEvent(_ context.Context, u string, e domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if events := s.events[u]; len(events) > 0 {
		e.Sequence = events[len(events)-1].Sequence + 1
	} else {
		e.Sequence = 1
	}
	s.events[u] = append(s.events[u], e)
	if len(s.events[u]) > maxEventsPerUser {
		s.events[u] = append([]domain.Event(nil), s.events[u][len(s.events[u])-maxEventsPerUser:]...)
	}
	return nil
}
func (s *Store) EventsSince(_ context.Context, u string, seq int64) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Event
	for _, e := range s.events[u] {
		if e.Sequence > seq {
			out = append(out, e)
		}
	}
	return out, nil
}
func (s *Store) EventBounds(_ context.Context, u string) (int64, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.events[u]
	if len(events) == 0 {
		return 0, 0, nil
	}
	return events[0].Sequence, events[len(events)-1].Sequence, nil
}
func (s *Store) GetQuiz(_ context.Context, u, id string) (domain.Quiz, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.quizzes[key(u, id)]
	return x, ok, nil
}
func (s *Store) SaveQuiz(_ context.Context, u string, q domain.Quiz) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quizzes[key(u, q.ID)] = q
	return nil
}
func (s *Store) SaveQuizResult(_ context.Context, u, k string, r domain.QuizResult) (domain.QuizResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.results[key(u, k)]
	if ok {
		return x, false, nil
	}
	s.results[key(u, k)] = r
	return r, true, nil
}
func (s *Store) CommitQuizAttempt(_ context.Context, userID, scope, fingerprint string, result domain.QuizResult, observations []ports.KnowledgeObservation) (domain.QuizResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	attemptKey := key(userID, scope)
	if saved, ok := s.results[attemptKey]; ok {
		if s.resultInputs[attemptKey] != fingerprint {
			return domain.QuizResult{}, false, ports.ErrIdempotencyConflict
		}
		return saved, false, nil
	}
	user, ok := s.users[userID]
	if !ok {
		return domain.QuizResult{}, false, errors.New("user not found")
	}

	updated := make(map[string]domain.KnowledgeNode, len(observations))
	result.MasteryChanges = make([]domain.MasteryChange, 0, len(observations))
	for _, observation := range observations {
		nodeKey := key(userID, observation.Topic)
		node, exists := updated[nodeKey]
		if !exists {
			node, exists = s.nodes[nodeKey]
		}
		if !exists {
			node = domain.KnowledgeNode{ID: observation.NodeID, UserID: userID, Domain: observation.Domain, TopicPath: observation.Topic, EaseFactor: 2.5}
		}
		before := node.Mastery
		node.Mastery = domain.UpdateMastery(node.Mastery, float64(observation.Quality)*20, node.Attempts)
		node.Confidence = math.Max(0, math.Min(100, observation.Confidence*100))
		node = domain.ScheduleSM2(node, observation.Quality, observation.At)
		updated[nodeKey] = node
		result.MasteryChanges = append(result.MasteryChanges, domain.MasteryChange{Topic: observation.Topic, Before: before, After: node.Mastery, NextRevisionDue: node.NextRevisionDue})
	}

	for nodeKey, node := range updated {
		s.nodes[nodeKey] = node
	}
	user.XP += result.XPAwarded
	s.users[userID] = user
	s.users[user.Subject] = user
	s.xpKeys[key(userID, "quiz:"+scope)] = true
	s.results[attemptKey] = result
	s.resultInputs[attemptKey] = fingerprint
	return result, true, nil
}
func (s *Store) KnowledgeNode(_ context.Context, u, t string) (domain.KnowledgeNode, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.nodes[key(u, t)]
	return x, ok, nil
}
func (s *Store) SaveKnowledgeNode(_ context.Context, n domain.KnowledgeNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[key(n.UserID, n.TopicPath)] = n
	return nil
}
func (s *Store) AddXP(_ context.Context, u, k string, n int) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.xpKeys[key(u, k)] {
		return s.users[u].XP, false, nil
	}
	x, ok := s.users[u]
	if !ok {
		return 0, false, errors.New("user not found")
	}
	x.XP += n
	s.users[u] = x
	s.users[x.Subject] = x
	s.xpKeys[key(u, k)] = true
	return x.XP, true, nil
}
func (s *Store) RecordSessionCompletion(_ context.Context, userID, sessionID string, at time.Time, minutes, xp int) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return 0, false, errors.New("user not found")
	}
	if s.completionKeys[key(userID, sessionID)] {
		return user.CurrentStreak, false, nil
	}
	location, err := time.LoadLocation(user.Preferences.Timezone)
	if err != nil {
		location = time.UTC
	}
	local := at.In(location)
	date := local.Format("2006-01-02")
	previous := local.AddDate(0, 0, -1).Format("2006-01-02")
	if s.activity[userID] == nil {
		s.activity[userID] = map[string]domain.ActivityDay{}
	}
	if _, alreadyActiveToday := s.activity[userID][date]; !alreadyActiveToday {
		if _, continued := s.activity[userID][previous]; continued {
			user.CurrentStreak++
		} else {
			user.CurrentStreak = 1
		}
	}
	day := s.activity[userID][date]
	day.Date = date
	day.StudyMinutes += minutes
	day.XP += xp
	s.activity[userID][date] = day
	s.completionKeys[key(userID, sessionID)] = true
	s.users[userID] = user
	s.users[user.Subject] = user
	return user.CurrentStreak, true, nil
}
func (s *Store) Activity(_ context.Context, userID string) ([]domain.ActivityDay, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	days := make([]domain.ActivityDay, 0, len(s.activity[userID]))
	for _, day := range s.activity[userID] {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return days, nil
}
func (s *Store) SaveDocument(_ context.Context, d domain.Document, c []domain.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[key(d.UserID, d.ID)] = d
	s.chunks = append(s.chunks, c...)
	return nil
}
func (s *Store) Documents(_ context.Context, u string) ([]domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var x []domain.Document
	for _, d := range s.documents {
		if d.UserID == u {
			x = append(x, d)
		}
	}
	sort.Slice(x, func(i, j int) bool { return x[i].CreatedAt.After(x[j].CreatedAt) })
	return x, nil
}
func (s *Store) Document(_ context.Context, u, id string) (domain.Document, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.documents[key(u, id)]
	return x, ok, nil
}
func (s *Store) DeleteDocument(_ context.Context, u, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, key(u, id))
	var c []domain.Chunk
	for _, x := range s.chunks {
		if !(x.UserID == u && x.DocumentID == id) {
			c = append(c, x)
		}
	}
	s.chunks = c
	return nil
}
func (s *Store) SearchChunks(_ context.Context, u, q string, limit int) ([]domain.Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	terms := strings.Fields(strings.ToLower(q))
	type scored struct {
		c domain.Chunk
		n int
	}
	var all []scored
	for _, c := range s.chunks {
		if c.UserID != u {
			continue
		}
		for _, t := range terms {
			if strings.Contains(strings.ToLower(c.Text), t) {
				for i := range all {
					_ = i
				}
			}
		}
		n := 0
		for _, t := range terms {
			n += strings.Count(strings.ToLower(c.Text), t)
		}
		if n > 0 {
			all = append(all, scored{c, n})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]domain.Chunk, len(all))
	for i := range all {
		out[i] = all[i].c
	}
	return out, nil
}
func (s *Store) SaveInterview(_ context.Context, x domain.Interview) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interviews[key(x.UserID, x.ID)] = cloneInterview(x)
	return nil
}
func (s *Store) Interview(_ context.Context, u, id string) (domain.Interview, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.interviews[key(u, id)]
	return cloneInterview(x), ok, nil
}
func (s *Store) CompareAndSwapInterview(_ context.Context, userID, interviewID string, expectedSequence int64, next domain.Interview) (domain.Interview, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	interviewKey := key(userID, interviewID)
	current, ok := s.interviews[interviewKey]
	if !ok {
		return domain.Interview{}, false, errors.New("interview not found")
	}
	if current.Sequence != expectedSequence {
		return cloneInterview(current), false, nil
	}
	if next.UserID != userID || next.ID != interviewID {
		return cloneInterview(current), false, errors.New("interview identity cannot change")
	}
	next = cloneInterview(next)
	s.interviews[interviewKey] = next
	return cloneInterview(next), true, nil
}
func (s *Store) KnowledgeGraph(_ context.Context, u string) ([]domain.KnowledgeNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var x []domain.KnowledgeNode
	for _, n := range s.nodes {
		if n.UserID == u {
			x = append(x, n)
		}
	}
	return x, nil
}

var _ = time.Time{}

func cloneInterview(x domain.Interview) domain.Interview {
	x.Messages = append([]domain.InterviewMessage(nil), x.Messages...)
	if x.ProcessedEvents != nil {
		x.ProcessedEvents = make(map[string]domain.InterviewEventReceipt, len(x.ProcessedEvents))
		for eventID, receipt := range x.ProcessedEvents {
			x.ProcessedEvents[eventID] = receipt
		}
	}
	if x.Scorecard != nil {
		scorecard := *x.Scorecard
		scorecard.Strengths = append([]string(nil), x.Scorecard.Strengths...)
		scorecard.Improvements = append([]string(nil), x.Scorecard.Improvements...)
		x.Scorecard = &scorecard
	}
	return x
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
