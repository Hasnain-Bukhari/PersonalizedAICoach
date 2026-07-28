package api

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/personalized-ai-coach/backend/internal/adapters/identity"
	"github.com/personalized-ai-coach/backend/internal/application"
	"github.com/personalized-ai-coach/backend/internal/domain"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type contextKey string

const userKey contextKey = "user"

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
			problem(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		u, err := a.Store.EnsureUser(r.Context(), claims.Subject, claims.Email)
		if err != nil {
			problem(w, 500, "identity_error", err.Error())
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
	jsonResponse(w, 200, map[string]string{"status": "ok"})
}
func (a *Server) daily(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		problem(w, 400, "invalid_date", err.Error())
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
	x, ok, e := a.Store.GetSession(r.Context(), user(r).ID, r.PathValue("id"))
	if e != nil || !ok {
		problem(w, 404, "not_found", "session not found")
		return
	}
	x.Status = "completed"
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
		Answers        []domain.Answer `json:"answers"`
		IdempotencyKey string          `json:"idempotency_key"`
	}
	if !decode(w, r, &in) {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = in.IdempotencyKey
	}
	x, e := a.Coach.SubmitQuiz(r.Context(), user(r).ID, r.PathValue("id"), key, in.Answers)
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
		problem(w, 400, "validation_error", "prompt is required")
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
		problem(w, 400, "invalid_upload", e.Error())
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
		problem(w, 400, "invalid_upload", e.Error())
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
	if e != nil || !ok {
		problem(w, 404, "not_found", "document not found")
		return
	}
	jsonResponse(w, 200, x)
}
func (a *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
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
	respond(w, days, err)
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
		problem(w, 400, "validation_error", "session_minutes must be between 10 and 240")
		return
	}
	if _, e := time.LoadLocation(p.Timezone); e != nil {
		problem(w, 400, "validation_error", "invalid timezone")
		return
	}
	u, e := a.Store.SavePreferences(r.Context(), user(r).ID, p)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, u.Preferences)
}
func (a *Server) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "streaming_unsupported", "response streaming unavailable")
		return
	}
	seq, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		events, e := a.Store.EventsSince(r.Context(), user(r).ID, seq)
		if e != nil {
			return
		}
		for _, x := range events {
			b, _ := json.Marshal(x)
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", x.Sequence, x.Type, b)
			seq = x.Sequence
		}
		f.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *Server) interviewWS(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("interview_id")
	if _, ok, _ := a.Store.Interview(r.Context(), user(r).ID, id); !ok {
		problem(w, 404, "not_found", "interview not found")
		return
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		problem(w, 426, "upgrade_required", "WebSocket upgrade required")
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
	accept := wsAccept(r.Header.Get("Sec-WebSocket-Key"))
	selectedProtocol := ""
	if strings.Contains(r.Header.Get("Sec-WebSocket-Protocol"), "coach.v1") {
		selectedProtocol = "Sec-WebSocket-Protocol: coach.v1\r\n"
	}
	fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n%s\r\n", accept, selectedProtocol)
	_ = rw.Flush()
	for {
		payload, e := readFrame(rw.Reader)
		if e != nil {
			return
		}
		var in struct {
			Content string `json:"content"`
			Payload struct {
				Content string `json:"content"`
			} `json:"payload"`
		}
		if json.Unmarshal(payload, &in) != nil {
			continue
		}
		if in.Content == "" {
			in.Content = in.Payload.Content
		}
		if strings.TrimSpace(in.Content) == "" {
			continue
		}
		x, e := a.Coach.InterviewReply(r.Context(), user(r).ID, id, in.Content)
		if e != nil {
			return
		}
		message := x.Messages[len(x.Messages)-1]
		b, _ := json.Marshal(domain.Event{ID: uuid(), Type: "interview.message", WorkflowID: id, Sequence: message.Sequence, Timestamp: message.At, Payload: message})
		if writeFrame(conn, b) != nil {
			return
		}
	}
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
func readFrame(r *bufio.Reader) ([]byte, error) {
	h := make([]byte, 2)
	if _, e := io.ReadFull(r, h); e != nil {
		return nil, e
	}
	if h[0]&15 == 8 {
		return nil, io.EOF
	}
	n := int(h[1] & 127)
	if n == 126 {
		b := make([]byte, 2)
		io.ReadFull(r, b)
		n = int(binary.BigEndian.Uint16(b))
	} else if n == 127 {
		return nil, errors.New("frame too large")
	}
	masked := h[1]&128 != 0
	if !masked {
		return nil, errors.New("client frame must be masked")
	}
	mask := make([]byte, 4)
	io.ReadFull(r, mask)
	p := make([]byte, n)
	if _, e := io.ReadFull(r, p); e != nil {
		return nil, e
	}
	for i := range p {
		p[i] ^= mask[i%4]
	}
	return p, nil
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

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		problem(w, 400, "invalid_json", e.Error())
		return false
	}
	return true
}
func respond(w http.ResponseWriter, v any, e error) {
	if e != nil {
		status, code := 500, "internal_error"
		msg := strings.ToLower(e.Error())
		if strings.Contains(msg, "not found") {
			status, code = 404, "not_found"
		} else if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "already scored") {
			status, code = 400, "validation_error"
		}
		problem(w, status, code, e.Error())
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
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status, "code": code, "detail": detail})
}
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = strconv.FormatInt(time.Now().UnixNano(), 36)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}
func recoverer(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				l.Error("request panic", "error", x)
				problem(w, 500, "internal_error", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
