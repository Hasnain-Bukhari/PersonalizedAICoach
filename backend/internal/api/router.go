package api

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/personalized-ai-coach/backend/internal/adapters/identity"
	"github.com/personalized-ai-coach/backend/internal/application"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"github.com/personalized-ai-coach/backend/internal/ports"
)

type contextKey string

const userKey contextKey = "user"

const (
	requestIDKey       contextKey = "request_id"
	maxWebSocketFrame             = 32 << 10
	webSocketLifetime             = time.Hour
	streamLifetime                = 30 * time.Minute
	streamWriteTimeout            = 10 * time.Second
)

type Server struct {
	Coach  *application.Coach
	Store  ports.Store
	Verify *identity.Verifier
	Log    *slog.Logger
}

func New(c *application.Coach, s ports.Store, v *identity.Verifier, l *slog.Logger) http.Handler {
	a := &Server{Coach: c, Store: s, Verify: v, Log: l}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.health)
	mux.HandleFunc("GET /api/v1/healthz", a.health)
	mux.HandleFunc("GET /api/v1/readyz", a.health)
	mux.HandleFunc("GET /api/v1/sessions/daily", a.daily)
	mux.HandleFunc("GET /api/v1/sessions/{id}", a.session)
	mux.HandleFunc("POST /api/v1/sessions/{id}/complete", a.complete)
	mux.HandleFunc("POST /api/v1/quiz/{id}/submit", a.quiz)
	mux.HandleFunc("POST /api/v1/interviews", a.createInterview)
	mux.HandleFunc("GET /api/v1/interviews/{id}", a.getInterview)
	mux.HandleFunc("GET /api/v1/interviews/{id}/scorecard", a.scorecard)
	mux.HandleFunc("GET /api/v1/interview/stream", a.interviewWS)
	mux.HandleFunc("POST /api/v1/knowledge/upload", a.upload)
	mux.HandleFunc("GET /api/v1/knowledge/documents", a.documents)
	mux.HandleFunc("GET /api/v1/knowledge/documents/{id}", a.document)
	mux.HandleFunc("DELETE /api/v1/knowledge/documents/{id}", a.deleteDocument)
	mux.HandleFunc("GET /api/v1/analytics/graph", a.graph)
	mux.HandleFunc("GET /api/v1/analytics/overview", a.overview)
	mux.HandleFunc("GET /api/v1/analytics/activity", a.activity)
	mux.HandleFunc("GET /api/v1/profile/preferences", a.getPreferences)
	mux.HandleFunc("PUT /api/v1/profile/preferences", a.putPreferences)
	mux.HandleFunc("GET /api/v1/events/stream", a.events)
	return requestID(recoverer(l, a.auth(mux)))
}

// WithCORS permits configured browser origins while keeping authorization on
// every application request. It is intentionally outside the auth middleware
// so browsers can complete an OPTIONS preflight without a bearer token.
func WithCORS(next http.Handler, origins []string) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; origin != "" && ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, Last-Event-ID, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if origin == "" {
				problem(w, http.StatusBadRequest, "invalid_cors_request", "Origin header required")
				return
			}
			if _, ok := allowed[origin]; !ok {
				problem(w, http.StatusForbidden, "origin_not_allowed", "Origin is not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/api/v1/healthz" || r.URL.Path == "/api/v1/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		token := bearerToken(r)
		if token == "" {
			problem(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
			return
		}
		claims, err := a.Verify.Verify(r.Context(), token)
		if err != nil {
			a.Log.Warn("authentication rejected", "request_id", requestIDFrom(r.Context()), "error", err)
			problem(w, http.StatusUnauthorized, "unauthorized", "Bearer token is invalid or expired")
			return
		}
		u, err := a.Store.EnsureUser(r.Context(), claims.Subject, claims.Email)
		if err != nil {
			a.Log.Error("identity lookup failed", "request_id", requestIDFrom(r.Context()), "error", err)
			problem(w, http.StatusServiceUnavailable, "identity_unavailable", "Identity service is unavailable")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

func bearerToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	// Browser WebSocket APIs cannot set Authorization. Carry the token in an
	// encoded subprotocol value so it is not exposed in query strings or access
	// logs. The server selects the stable coach.v1 application protocol below.
	if r.URL.Path == "/api/v1/interview/stream" {
		for _, protocol := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
			protocol = strings.TrimSpace(protocol)
			if !strings.HasPrefix(protocol, "auth.") {
				continue
			}
			decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(protocol, "auth."))
			if err == nil {
				return string(decoded)
			}
		}
	}
	return ""
}
func user(r *http.Request) domain.User { return r.Context().Value(userKey).(domain.User) }
func (a *Server) health(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, 200, map[string]string{"status": "ok", "service": "personalized-ai-coach-api"})
}
func (a *Server) daily(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		problem(w, http.StatusUnprocessableEntity, "invalid_date", "date must use YYYY-MM-DD")
		return
	}
	s, _, err := a.Coach.Daily(r.Context(), user(r).ID, date)
	respond(w, s, err)
}
func (a *Server) session(w http.ResponseWriter, r *http.Request) {
	x, ok, e := a.Store.GetSession(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !ok {
		problem(w, 404, "not_found", "session not found")
		return
	}
	jsonResponse(w, 200, x)
}
func (a *Server) complete(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confidence     *float64 `json:"confidence"`
		Reflection     string   `json:"reflection"`
		IdempotencyKey string   `json:"idempotency_key"`
	}
	if !decodeOptional(w, r, &in) {
		return
	}
	headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if headerKey != "" && in.IdempotencyKey != "" && headerKey != in.IdempotencyKey {
		problem(w, http.StatusConflict, "idempotency_conflict", "Idempotency keys in the header and body must match")
		return
	}
	key := headerKey
	if key == "" {
		key = strings.TrimSpace(in.IdempotencyKey)
	}
	if len(key) > 128 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "idempotency_key must not exceed 128 characters")
		return
	}
	if in.Confidence != nil && (*in.Confidence < 0 || *in.Confidence > 100) {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "confidence must be between 0 and 100")
		return
	}
	if len([]rune(in.Reflection)) > 4000 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "reflection must not exceed 4000 characters")
		return
	}
	x, ok, e := a.Store.GetSession(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !ok {
		problem(w, 404, "not_found", "session not found")
		return
	}
	x.Status = "completed"
	if strings.TrimSpace(in.Reflection) != "" {
		x.Reflection = strings.TrimSpace(in.Reflection)
	}
	x.UpdatedAt = time.Now().UTC()
	e = a.Store.SaveDailySession(r.Context(), x)
	if e == nil {
		_, _, e = a.Store.AddXP(r.Context(), x.UserID, "session:"+x.ID, 25)
	}
	if e == nil {
		_, _, e = a.Store.RecordSessionCompletion(r.Context(), x.UserID, x.ID, x.UpdatedAt, x.EstimatedMinutes, 25)
	}
	respond(w, x, e)
}
func (a *Server) quiz(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Answers []struct {
			QuestionID string   `json:"question_id"`
			Value      string   `json:"value"`
			Confidence *float64 `json:"confidence"`
		} `json:"answers"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &in) {
		return
	}
	headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	bodyKey := strings.TrimSpace(in.IdempotencyKey)
	if headerKey != "" && bodyKey != "" && headerKey != bodyKey {
		problem(w, http.StatusConflict, "idempotency_conflict", "Idempotency keys in the header and body must match")
		return
	}
	key := headerKey
	if key == "" {
		key = bodyKey
	}
	if key == "" || len(key) > 128 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "An idempotency key of at most 128 characters is required")
		return
	}
	if len(in.Answers) == 0 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "At least one answer is required")
		return
	}
	quiz, ok, err := a.Store.GetQuiz(r.Context(), user(r).ID, r.PathValue("id"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	if !ok {
		problem(w, http.StatusNotFound, "not_found", "quiz not found")
		return
	}
	questionIDs := make(map[string]struct{}, len(quiz.Questions))
	for _, question := range quiz.Questions {
		questionIDs[question.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(in.Answers))
	answers := make([]domain.Answer, 0, len(in.Answers))
	for _, answer := range in.Answers {
		if _, exists := questionIDs[answer.QuestionID]; !exists {
			problem(w, http.StatusUnprocessableEntity, "validation_error", "answer contains an unknown question_id")
			return
		}
		if _, duplicate := seen[answer.QuestionID]; duplicate {
			problem(w, http.StatusUnprocessableEntity, "validation_error", "answers must not contain duplicate question_id values")
			return
		}
		if strings.TrimSpace(answer.Value) == "" || answer.Confidence == nil || *answer.Confidence < 0 || *answer.Confidence > 1 {
			problem(w, http.StatusUnprocessableEntity, "validation_error", "each answer requires a value and confidence between 0 and 1")
			return
		}
		seen[answer.QuestionID] = struct{}{}
		answers = append(answers, domain.Answer{QuestionID: answer.QuestionID, Value: strings.TrimSpace(answer.Value), Confidence: *answer.Confidence})
	}
	x, e := a.Coach.SubmitQuiz(r.Context(), user(r).ID, r.PathValue("id"), key, answers)
	respond(w, x, e)
}
func (a *Server) createInterview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Prompt string `json:"prompt"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Prompt) == "" {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "prompt is required")
		return
	}
	if n := len([]rune(in.Prompt)); n < 3 || n > 1000 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "prompt must be between 3 and 1000 characters")
		return
	}
	x, e := a.Coach.CreateInterview(r.Context(), user(r).ID, in.Prompt)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 201, x)
}
func (a *Server) getInterview(w http.ResponseWriter, r *http.Request) {
	x, ok, e := a.Store.Interview(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !ok {
		problem(w, 404, "not_found", "interview not found")
		return
	}
	jsonResponse(w, 200, x)
}
func (a *Server) scorecard(w http.ResponseWriter, r *http.Request) {
	x, ok, e := a.Store.Interview(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil || !ok {
		problem(w, 404, "not_found", "interview not found")
		return
	}
	if x.Scorecard == nil {
		problem(w, 409, "not_ready", "interview is not scored")
		return
	}
	jsonResponse(w, 200, x.Scorecard)
}
func (a *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 26<<20)
	f, h, e := r.FormFile("file")
	if e != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(e, &tooLarge) || strings.Contains(strings.ToLower(e.Error()), "request body too large") {
			problem(w, http.StatusRequestEntityTooLarge, "file_too_large", "Upload exceeds the 25 MiB limit")
		} else {
			problem(w, http.StatusUnprocessableEntity, "invalid_upload", "A multipart file is required")
		}
		return
	}
	defer f.Close()
	ct := h.Header.Get("Content-Type")
	if ct == "" {
		ct = contentType(f)
		_, _ = f.Seek(0, io.SeekStart)
	}
	d, e := a.Coach.Upload(r.Context(), user(r).ID, h.Filename, ct, h.Size, f)
	if e != nil {
		message := strings.ToLower(e.Error())
		switch {
		case strings.Contains(message, "25 mib") || strings.Contains(message, "too large"):
			problem(w, http.StatusRequestEntityTooLarge, "file_too_large", "Upload exceeds the 25 MiB limit")
		case strings.Contains(message, "content type") || strings.Contains(message, "unsupported"):
			problem(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "File type is not supported")
		default:
			problem(w, http.StatusUnprocessableEntity, "invalid_upload", e.Error())
		}
		return
	}
	jsonResponse(w, 202, d)
}
func contentType(f multipart.File) string {
	b := make([]byte, 512)
	n, _ := f.Read(b)
	return http.DetectContentType(b[:n])
}
func (a *Server) documents(w http.ResponseWriter, r *http.Request) {
	x, e := a.Store.Documents(r.Context(), user(r).ID)
	respond(w, x, e)
}
func (a *Server) document(w http.ResponseWriter, r *http.Request) {
	x, ok, e := a.Store.Document(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !ok {
		problem(w, 404, "not_found", "document not found")
		return
	}
	jsonResponse(w, 200, x)
}
func (a *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	if _, ok, e := a.Store.Document(r.Context(), user(r).ID, r.PathValue("id")); e != nil {
		respond(w, nil, e)
		return
	} else if !ok {
		problem(w, http.StatusNotFound, "not_found", "document not found")
		return
	}
	if e := a.Store.DeleteDocument(r.Context(), user(r).ID, r.PathValue("id")); e != nil {
		respond(w, nil, e)
		return
	}
	w.WriteHeader(204)
}
func (a *Server) graph(w http.ResponseWriter, r *http.Request) {
	x, e := a.Store.KnowledgeGraph(r.Context(), user(r).ID)
	respond(w, map[string]any{"nodes": x, "edges": []any{}}, e)
}
func (a *Server) overview(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	nodes, e := a.Store.KnowledgeGraph(r.Context(), u.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	avg := 0.0
	for _, n := range nodes {
		avg += n.Mastery
	}
	if len(nodes) > 0 {
		avg /= float64(len(nodes))
	}
	jsonResponse(w, 200, map[string]any{"xp": u.XP, "streak": u.CurrentStreak, "mastery": avg, "exam_readiness": avg})
}
func (a *Server) activity(w http.ResponseWriter, r *http.Request) {
	days, err := a.Store.Activity(r.Context(), user(r).ID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	from, ok := parseOptionalDate(w, r.URL.Query().Get("from"), "from")
	if !ok {
		return
	}
	to, ok := parseOptionalDate(w, r.URL.Query().Get("to"), "to")
	if !ok {
		return
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		problem(w, http.StatusUnprocessableEntity, "invalid_date_range", "from must be on or before to")
		return
	}
	filtered := make([]domain.ActivityDay, 0, len(days))
	for _, day := range days {
		date, parseErr := time.Parse("2006-01-02", day.Date)
		if parseErr != nil {
			continue
		}
		if (!from.IsZero() && date.Before(from)) || (!to.IsZero() && date.After(to)) {
			continue
		}
		filtered = append(filtered, day)
	}
	jsonResponse(w, http.StatusOK, filtered)
}
func (a *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, 200, user(r).Preferences)
}
func (a *Server) putPreferences(w http.ResponseWriter, r *http.Request) {
	var p domain.Preferences
	if !decode(w, r, &p) {
		return
	}
	if p.SessionMinutes < 10 || p.SessionMinutes > 240 {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "session_minutes must be between 10 and 240")
		return
	}
	if _, e := time.LoadLocation(p.Timezone); e != nil {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "invalid timezone")
		return
	}
	if _, e := time.Parse("15:04", p.DailyTime); e != nil {
		problem(w, http.StatusUnprocessableEntity, "validation_error", "daily_time must use HH:MM")
		return
	}
	u, e := a.Store.SavePreferences(r.Context(), user(r).ID, p)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, u.Preferences)
}

func parseOptionalDate(w http.ResponseWriter, value, name string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, "invalid_date", name+" must use YYYY-MM-DD")
		return time.Time{}, false
	}
	return parsed, true
}
func (a *Server) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "streaming_unsupported", "response streaming unavailable")
		return
	}
	seq := int64(0)
	cursorSupplied := strings.TrimSpace(r.Header.Get("Last-Event-ID")) != ""
	if cursor := strings.TrimSpace(r.Header.Get("Last-Event-ID")); cursor != "" {
		var err error
		seq, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil || seq < 0 {
			problem(w, http.StatusUnprocessableEntity, "invalid_cursor", "Last-Event-ID must be a non-negative integer")
			return
		}
	}
	oldest, latest, err := a.Store.EventBounds(r.Context(), user(r).ID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if cursorSupplied && oldest > 0 && seq < oldest-1 {
		problem(w, http.StatusConflict, "cursor_expired", "The event cursor is no longer retained; reconnect without Last-Event-ID to resynchronize")
		return
	}
	if cursorSupplied && seq > latest {
		problem(w, http.StatusConflict, "cursor_ahead", "The event cursor is ahead of the stream; reconnect without Last-Event-ID to resynchronize")
		return
	}
	controller := http.NewResponseController(w)
	if err = controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil {
		problem(w, http.StatusInternalServerError, "streaming_unsupported", "Response deadlines are unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "retry: 2000\n\n")
	f.Flush()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lifetime := time.NewTimer(streamLifetime)
	defer lifetime.Stop()
	for {
		events, e := a.Store.EventsSince(r.Context(), user(r).ID, seq)
		if e != nil {
			return
		}
		for _, x := range events {
			b, _ := json.Marshal(x)
			if _, e = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", x.Sequence, x.Type, b); e != nil {
				return
			}
			seq = x.Sequence
		}
		if len(events) == 0 {
			if _, e = io.WriteString(w, ": keep-alive\n\n"); e != nil {
				return
			}
		}
		if e = controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); e != nil {
			return
		}
		f.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-lifetime.C:
			return
		case <-ticker.C:
		}
	}
}

func (a *Server) interviewWS(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("interview_id")
	interview, ok, err := a.Store.Interview(r.Context(), user(r).ID, id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if !ok {
		problem(w, 404, "not_found", "interview not found")
		return
	}
	afterSequence := int64(0)
	resumeRequested := r.URL.Query().Has("after_sequence")
	if resumeRequested {
		afterSequence, err = strconv.ParseInt(r.URL.Query().Get("after_sequence"), 10, 64)
		if err != nil || afterSequence < 0 {
			problem(w, http.StatusUnprocessableEntity, "invalid_cursor", "after_sequence must be a non-negative integer")
			return
		}
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || !headerHasToken(r.Header.Get("Connection"), "upgrade") {
		problem(w, 426, "upgrade_required", "WebSocket upgrade required")
		return
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" || !validWebSocketKey(r.Header.Get("Sec-WebSocket-Key")) {
		problem(w, http.StatusBadRequest, "invalid_websocket_handshake", "A valid WebSocket version and key are required")
		return
	}
	if !headerHasToken(r.Header.Get("Sec-WebSocket-Protocol"), "coach.v1") {
		problem(w, http.StatusBadRequest, "unsupported_websocket_protocol", "The coach.v1 WebSocket protocol is required")
		return
	}
	h, ok := w.(http.Hijacker)
	if !ok {
		problem(w, 500, "websocket_unsupported", "hijacking unavailable")
		return
	}
	conn, rw, e := h.Hijack()
	if e != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(webSocketLifetime))
	accept := wsAccept(r.Header.Get("Sec-WebSocket-Key"))
	fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\nSec-WebSocket-Protocol: coach.v1\r\n\r\n", accept)
	_ = rw.Flush()
	if resumeRequested {
		for _, message := range interview.Messages {
			if message.Sequence > afterSequence {
				if writeInterviewEvent(conn, id, message) != nil {
					return
				}
			}
		}
	}
	for {
		frame, frameErr := readFrame(rw.Reader)
		if frameErr != nil {
			return
		}
		switch frame.opcode {
		case 0x8:
			_ = writeControlFrame(conn, 0x8, frame.payload)
			return
		case 0x9:
			if writeControlFrame(conn, 0xA, frame.payload) != nil {
				return
			}
			continue
		case 0xA:
			continue
		case 0x1:
		default:
			_ = writeCloseFrame(conn, 1003, "text frames required")
			return
		}
		var in struct {
			EventID     string `json:"event_id"`
			Type        string `json:"type"`
			InterviewID string `json:"interview_id"`
			Sequence    int64  `json:"sequence"`
			Timestamp   string `json:"timestamp"`
			Payload struct {
				Content string `json:"content"`
			} `json:"payload"`
		}
		decoder := json.NewDecoder(bytes.NewReader(frame.payload))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&in) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			_ = writeCloseFrame(conn, 1007, "invalid event JSON")
			return
		}
		if in.EventID == "" || in.Type != "candidate.message" || in.InterviewID != id || in.Sequence < 1 || strings.TrimSpace(in.Timestamp) == "" {
			_ = writeCloseFrame(conn, 1008, "invalid event envelope")
			return
		}
		if len([]rune(in.Payload.Content)) > 20000 || strings.TrimSpace(in.Payload.Content) == "" {
			_ = writeCloseFrame(conn, 1008, "content is required and must not exceed 20000 characters")
			return
		}
		x, applied, e := a.Coach.InterviewReplyEvent(r.Context(), user(r).ID, id, in.EventID, in.Sequence, strings.TrimSpace(in.Payload.Content))
		if e != nil {
			if errors.Is(e, ports.ErrInterviewSequenceConflict) {
				if writeCandidateAck(conn, id, in.EventID, x, false, false, "sequence_conflict") != nil {
					return
				}
				continue
			}
			if errors.Is(e, ports.ErrIdempotencyConflict) {
				_ = writeCloseFrame(conn, 1008, "event_id conflicts with a previous message")
			} else {
				_ = writeCloseFrame(conn, 1011, "unable to process interview response")
			}
			return
		}
		if writeCandidateAck(conn, id, in.EventID, x, true, applied, "") != nil {
			return
		}
		if !applied {
			continue
		}
		message := x.Messages[len(x.Messages)-1]
		if writeInterviewEvent(conn, id, message) != nil {
			return
		}
	}
}

func writeCandidateAck(conn net.Conn, interviewID, submittedEventID string, interview domain.Interview, accepted, applied bool, reason string) error {
	payload := map[string]any{
		"submitted_event_id": submittedEventID,
		"accepted":           accepted,
		"applied":            applied,
		"next_sequence":      interview.LastClientSequence + 1,
	}
	if reason != "" {
		payload["reason"] = reason
	}
	sequence := interview.Sequence
	if sequence < 1 {
		sequence = 1
	}
	body, err := json.Marshal(domain.Event{ID: uuid(), Type: "candidate.ack", WorkflowID: interviewID, Sequence: sequence, Timestamp: time.Now().UTC(), Payload: payload})
	if err != nil {
		return err
	}
	return writeFrame(conn, body)
}

func headerHasToken(value, want string) bool {
	for _, token := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(token), want) {
			return true
		}
	}
	return false
}

func validWebSocketKey(value string) bool {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return err == nil && len(decoded) == 16
}

func writeInterviewEvent(conn net.Conn, interviewID string, message domain.InterviewMessage) error {
	payload, err := json.Marshal(domain.Event{ID: uuid(), Type: "interview.message", WorkflowID: interviewID, Sequence: message.Sequence, Timestamp: message.At, Payload: message})
	if err != nil {
		return err
	}
	return writeFrame(conn, payload)
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
func wsAccept(k string) string {
	sum := sha1.Sum([]byte(k + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}
type webSocketFrame struct {
	opcode  byte
	payload []byte
}

func readFrame(r *bufio.Reader) (webSocketFrame, error) {
	h := make([]byte, 2)
	if _, e := io.ReadFull(r, h); e != nil {
		return webSocketFrame{}, e
	}
	if h[0]&0x70 != 0 || h[0]&0x80 == 0 {
		return webSocketFrame{}, errors.New("fragmented or reserved WebSocket frame")
	}
	opcode := h[0] & 15
	n := int(h[1] & 127)
	if n == 126 {
		b := make([]byte, 2)
		if _, err := io.ReadFull(r, b); err != nil {
			return webSocketFrame{}, err
		}
		n = int(binary.BigEndian.Uint16(b))
	} else if n == 127 {
		return webSocketFrame{}, errors.New("frame too large")
	}
	if n > maxWebSocketFrame || (opcode >= 0x8 && n > 125) {
		return webSocketFrame{}, errors.New("frame too large")
	}
	masked := h[1]&128 != 0
	if !masked {
		return webSocketFrame{}, errors.New("client frame must be masked")
	}
	mask := make([]byte, 4)
	if _, err := io.ReadFull(r, mask); err != nil {
		return webSocketFrame{}, err
	}
	p := make([]byte, n)
	if _, e := io.ReadFull(r, p); e != nil {
		return webSocketFrame{}, e
	}
	for i := range p {
		p[i] ^= mask[i%4]
	}
	return webSocketFrame{opcode: opcode, payload: p}, nil
}
func writeFrame(w net.Conn, p []byte) error {
	h := []byte{0x81}
	if len(p) < 126 {
		h = append(h, byte(len(p)))
	} else {
		h = append(h, 126, byte(len(p)>>8), byte(len(p)))
	}
	if _, e := w.Write(h); e != nil {
		return e
	}
	_, e := w.Write(p)
	return e
}

func writeControlFrame(w net.Conn, opcode byte, payload []byte) error {
	if len(payload) > 125 {
		return errors.New("control frame too large")
	}
	if _, err := w.Write(append([]byte{0x80 | opcode, byte(len(payload))}, payload...)); err != nil {
		return err
	}
	return nil
}

func writeCloseFrame(w net.Conn, code int, reason string) error {
	payload := make([]byte, 2, 125)
	binary.BigEndian.PutUint16(payload, uint16(code))
	payload = append(payload, []byte(reason)...)
	if len(payload) > 125 {
		payload = payload[:125]
	}
	return writeControlFrame(w, 0x8, payload)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		problem(w, http.StatusBadRequest, "invalid_json", "Request body must contain valid JSON")
		return false
	}
	if e := d.Decode(&struct{}{}); e != io.EOF {
		problem(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON value")
		return false
	}
	return true
}

func decodeOptional(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	return decode(w, r, v)
}

func respond(w http.ResponseWriter, v any, e error) {
	if e != nil {
		status, code := 500, "internal_error"
		msg := strings.ToLower(e.Error())
		if errors.Is(e, ports.ErrIdempotencyConflict) {
			status, code = http.StatusConflict, "idempotency_conflict"
		} else if strings.Contains(msg, "not found") {
			status, code = 404, "not_found"
		} else if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "already scored") {
			status, code = http.StatusUnprocessableEntity, "validation_error"
		}
		detail := e.Error()
		if status >= 500 {
			detail = "Unexpected server error"
		}
		problem(w, status, code, detail)
		return
	}
	jsonResponse(w, 200, v)
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	payload := map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status, "code": code, "detail": detail}
	if id := w.Header().Get("X-Request-ID"); id != "" {
		payload["trace_id"] = id
	}
	_ = json.NewEncoder(w).Encode(payload)
}
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if len(id) > 128 || strings.ContainsAny(id, "\r\n\t") {
			id = ""
		}
		if id == "" {
			id = uuid()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
func recoverer(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				l.Error("request panic", "request_id", requestIDFrom(r.Context()), "error", x)
				problem(w, 500, "internal_error", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
