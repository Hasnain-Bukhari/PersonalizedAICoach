package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/personalized-ai-coach/backend/internal/adapters/identity"
	"github.com/personalized-ai-coach/backend/internal/adapters/llm"
	"github.com/personalized-ai-coach/backend/internal/adapters/memory"
	"github.com/personalized-ai-coach/backend/internal/application"
	"github.com/personalized-ai-coach/backend/internal/domain"
)

func handler() http.Handler {
	h, _ := testHandler()
	return h
}
func testHandler() (http.Handler, *memory.Store) {
	s := memory.New()
	return New(application.New(s, llm.Fake{}), s, identity.New("https://example.invalid/", "coach", "dev"), slog.New(slog.NewTextHandler(io.Discard, nil))), s
}
func TestHealthAndAuthentication(t *testing.T) {
	h := handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatalf("health=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/sessions/daily", nil))
	if w.Code != 401 {
		t.Fatalf("unauthorized=%d", w.Code)
	}
}

func TestProblemDetailsAreSafeAndTraceable(t *testing.T) {
	h := handler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/daily?date=bad", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	r.Header.Set("X-Request-ID", "request-123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out["trace_id"] != "request-123" || out["code"] != "invalid_date" {
		t.Fatalf("problem=%s", w.Body.String())
	}

	r = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/daily", nil)
	r.Header.Set("Authorization", "Bearer malformed")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "malformed JWT") || !strings.Contains(w.Body.String(), "invalid or expired") {
		t.Fatalf("authentication detail leaked implementation error: %s", w.Body.String())
	}
}

func TestUploadAndDeleteStatusMappings(t *testing.T) {
	h := handler()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "payload.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("MZ executable"))
	_ = writer.Close()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/upload", &body)
	r.Header.Set("Authorization", "Bearer dev:alice")
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("upload status=%d body=%s", w.Code, w.Body.String())
	}

	r = httptest.NewRequest(http.MethodDelete, "/api/v1/knowledge/documents/missing", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestActivityDateFilteringAndValidation(t *testing.T) {
	h, store := testHandler()
	ctx := context.Background()
	u, err := store.EnsureUser(ctx, "alice", "alice@example.test")
	if err != nil {
		t.Fatal(err)
	}
	for i, date := range []string{"2026-07-01", "2026-07-02", "2026-07-03"} {
		at, _ := time.Parse("2006-01-02", date)
		if _, _, err = store.RecordSessionCompletion(ctx, u.ID, string(rune('a'+i)), at, 10, 5); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/activity?from=2026-07-02&to=2026-07-02", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || strings.Count(w.Body.String(), "study_minutes") != 1 || !strings.Contains(w.Body.String(), "2026-07-02") {
		t.Fatalf("filtered status=%d body=%s", w.Code, w.Body.String())
	}

	r = httptest.NewRequest(http.MethodGet, "/api/v1/analytics/activity?from=2026-07-03&to=2026-07-01", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("range status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSSECursorAndWebSocketHandshakeValidation(t *testing.T) {
	h := handler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	r.Header.Set("Last-Event-ID", "not-a-number")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("SSE status=%d body=%s", w.Code, w.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/api/v1/interviews", strings.NewReader(`{"prompt":"Design a queue"}`))
	create.Header.Set("Authorization", "Bearer dev:alice")
	create.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, create)
	var interview struct{ ID string `json:"id"` }
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &interview) != nil {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/interview/stream?interview_id="+interview.ID, nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_websocket_handshake") {
		t.Fatalf("websocket status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuizRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       func(domain.DailySession) string
		headerKey  string
		wantStatus int
	}{
		{"mismatched idempotency keys", func(s domain.DailySession) string { return fmt.Sprintf(`{"idempotency_key":"body","answers":[{"question_id":%q,"value":"answer","confidence":0.5}]}`, s.Quiz.Questions[0].ID) }, "header", http.StatusConflict},
		{"missing idempotency key", func(s domain.DailySession) string { return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"answer","confidence":0.5}]}`, s.Quiz.Questions[0].ID) }, "", http.StatusUnprocessableEntity},
		{"oversized idempotency key", func(s domain.DailySession) string { return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"answer","confidence":0.5}]}`, s.Quiz.Questions[0].ID) }, strings.Repeat("x", 129), http.StatusUnprocessableEntity},
		{"empty answers", func(domain.DailySession) string { return `{"answers":[]}` }, "key", http.StatusUnprocessableEntity},
		{"unknown question", func(domain.DailySession) string { return `{"answers":[{"question_id":"00000000-0000-4000-8000-000000000000","value":"answer","confidence":0.5}]}` }, "key", http.StatusUnprocessableEntity},
		{"duplicate question", func(s domain.DailySession) string { return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"one","confidence":0.5},{"question_id":%q,"value":"two","confidence":0.5}]}`, s.Quiz.Questions[0].ID, s.Quiz.Questions[0].ID) }, "key", http.StatusUnprocessableEntity},
		{"missing answer confidence", func(s domain.DailySession) string { return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"answer"}]}`, s.Quiz.Questions[0].ID) }, "key", http.StatusUnprocessableEntity},
		{"invalid answer confidence", func(s domain.DailySession) string { return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"answer","confidence":1.1}]}`, s.Quiz.Questions[0].ID) }, "key", http.StatusUnprocessableEntity},
		{"unsupported top-level confidence", func(s domain.DailySession) string { return fmt.Sprintf(`{"confidence":0.8,"answers":[{"question_id":%q,"value":"answer","confidence":0.5}]}`, s.Quiz.Questions[0].ID) }, "key", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler()
			daily := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/daily?date=2026-07-29", nil)
			daily.Header.Set("Authorization", "Bearer dev:quiz-user")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, daily)
			var session domain.DailySession
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &session) != nil {
				t.Fatalf("daily status=%d body=%s", w.Code, w.Body.String())
			}
			r := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/"+session.Quiz.ID+"/submit", strings.NewReader(tc.body(session)))
			r.Header.Set("Authorization", "Bearer dev:quiz-user")
			r.Header.Set("Content-Type", "application/json")
			if tc.headerKey != "" {
				r.Header.Set("Idempotency-Key", tc.headerKey)
			}
			w = httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestSSERejectsExpiredAndAheadCursors(t *testing.T) {
	h, store := testHandler()
	ctx := context.Background()
	u, err := store.EnsureUser(ctx, "cursor-user", "cursor@example.test")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1002; i++ {
		if err = store.PublishEvent(ctx, u.ID, domain.Event{ID: fmt.Sprintf("event-%d", i), Type: "workflow.planning", Timestamp: time.Now().UTC(), Payload: map[string]any{}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		cursor, code string
	}{
		{"0", "cursor_expired"},
		{"1003", "cursor_ahead"},
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
		r.Header.Set("Authorization", "Bearer dev:cursor-user")
		r.Header.Set("Last-Event-ID", tc.cursor)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), tc.code) || !strings.Contains(w.Body.String(), "resynchronize") {
			t.Fatalf("cursor=%s status=%d body=%s", tc.cursor, w.Code, w.Body.String())
		}
	}
}
func TestCORSPreflight(t *testing.T) {
	h := WithCORS(handler(), []string{"http://localhost:5173"})
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/daily", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	r.Header.Set("Access-Control-Request-Method", http.MethodGet)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("preflight status=%d origin=%q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}

	r = httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/daily", nil)
	r.Header.Set("Origin", "https://malicious.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disallowed preflight status=%d origin=%q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}
func TestDailySessionAPI(t *testing.T) {
	h := handler()
	r := httptest.NewRequest("GET", "/api/v1/sessions/daily?date=2026-07-26", nil)
	r.Header.Set("Authorization", "Bearer dev:alice::alice@example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out["status"] != "published" {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}
func TestTenantIsolation(t *testing.T) {
	h := handler()
	for _, subject := range []string{"alice", "bob"} {
		r := httptest.NewRequest("GET", "/api/v1/sessions/daily?date=2026-07-26", nil)
		r.Header.Set("Authorization", "Bearer dev:"+subject)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/analytics/graph", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}

type candidateAck struct {
	Type    string `json:"type"`
	Payload struct {
		SubmittedEventID string `json:"submitted_event_id"`
		Accepted         bool   `json:"accepted"`
		Applied          bool   `json:"applied"`
		NextSequence     int64  `json:"next_sequence"`
		Reason           string `json:"reason"`
	} `json:"payload"`
}

func TestInterviewWebSocketAcknowledgesRetriesAndSequenceConflicts(t *testing.T) {
	server := httptest.NewServer(handler())
	defer server.Close()

	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/interviews", strings.NewReader(`{"prompt":"Design a durable queue"}`))
	request.Header.Set("Authorization", "Bearer dev:ws-user")
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var interview domain.Interview
	if response.StatusCode != http.StatusCreated || json.NewDecoder(response.Body).Decode(&interview) != nil {
		t.Fatalf("create interview status=%d", response.StatusCode)
	}

	firstConn, firstReader := openTestWebSocket(t, server.URL, interview.ID)
	const (
		firstEventID  = "11111111-1111-4111-8111-111111111111"
		secondEventID = "22222222-2222-4222-8222-222222222222"
		thirdEventID  = "33333333-3333-4333-8333-333333333333"
	)
	firstEvent := candidateEvent(interview.ID, firstEventID, 1, "first answer")
	writeMaskedText(t, firstConn, firstEvent)
	ack := readCandidateAck(t, firstReader)
	if !ack.Payload.Accepted || !ack.Payload.Applied || ack.Payload.SubmittedEventID != firstEventID || ack.Payload.NextSequence != 2 {
		t.Fatalf("applied ack=%+v", ack)
	}
	if event := readServerEvent(t, firstReader); event.Type != "interview.message" {
		t.Fatalf("expected interviewer message after ack, got %q", event.Type)
	}
	_ = firstConn.Close()

	secondConn, secondReader := openTestWebSocket(t, server.URL, interview.ID)
	defer secondConn.Close()
	writeMaskedText(t, secondConn, firstEvent)
	ack = readCandidateAck(t, secondReader)
	if !ack.Payload.Accepted || ack.Payload.Applied || ack.Payload.SubmittedEventID != firstEventID || ack.Payload.NextSequence != 2 {
		t.Fatalf("retry ack=%+v", ack)
	}

	writeMaskedText(t, secondConn, candidateEvent(interview.ID, secondEventID, 1, "competing answer"))
	ack = readCandidateAck(t, secondReader)
	if ack.Payload.Accepted || ack.Payload.Applied || ack.Payload.Reason != "sequence_conflict" || ack.Payload.SubmittedEventID != secondEventID || ack.Payload.NextSequence != 2 {
		t.Fatalf("sequence conflict ack=%+v", ack)
	}

	writeMaskedText(t, secondConn, candidateEvent(interview.ID, thirdEventID, 2, "next answer"))
	ack = readCandidateAck(t, secondReader)
	if !ack.Payload.Accepted || !ack.Payload.Applied || ack.Payload.NextSequence != 3 {
		t.Fatalf("post-conflict ack=%+v", ack)
	}
	if event := readServerEvent(t, secondReader); event.Type != "interview.message" {
		t.Fatalf("expected connection to remain usable, got %q", event.Type)
	}
}

func openTestWebSocket(t *testing.T, serverURL, interviewID string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", strings.TrimPrefix(serverURL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	key := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, _ = fmt.Fprintf(conn, "GET /api/v1/interview/stream?interview_id=%s HTTP/1.1\r\nHost: %s\r\nAuthorization: Bearer dev:ws-user\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Protocol: coach.v1\r\n\r\n", interviewID, strings.TrimPrefix(serverURL, "http://"), key)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		t.Fatalf("websocket handshake status=%d", response.StatusCode)
	}
	return conn, reader
}

func candidateEvent(interviewID, eventID string, sequence int64, content string) []byte {
	payload, _ := json.Marshal(map[string]any{"event_id": eventID, "type": "candidate.message", "interview_id": interviewID, "sequence": sequence, "timestamp": time.Now().UTC().Format(time.RFC3339Nano), "payload": map[string]string{"content": content}})
	return payload
}

func writeMaskedText(t *testing.T, conn net.Conn, payload []byte) {
	t.Helper()
	header := []byte{0x81}
	if len(payload) < 126 {
		header = append(header, 0x80|byte(len(payload)))
	} else {
		header = append(header, 0x80|126, byte(len(payload)>>8), byte(len(payload)))
	}
	mask := []byte{1, 2, 3, 4}
	header = append(header, mask...)
	masked := append([]byte(nil), payload...)
	for i := range masked {
		masked[i] ^= mask[i%len(mask)]
	}
	if _, err := conn.Write(append(header, masked...)); err != nil {
		t.Fatal(err)
	}
}

func readCandidateAck(t *testing.T, reader *bufio.Reader) candidateAck {
	t.Helper()
	payload := readServerPayload(t, reader)
	var ack candidateAck
	if err := json.Unmarshal(payload, &ack); err != nil || ack.Type != "candidate.ack" {
		t.Fatalf("candidate ack=%s error=%v", payload, err)
	}
	return ack
}

func readServerEvent(t *testing.T, reader *bufio.Reader) domain.Event {
	t.Helper()
	payload := readServerPayload(t, reader)
	var event domain.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("server event=%s error=%v", payload, err)
	}
	return event
}

func readServerPayload(t *testing.T, reader *bufio.Reader) []byte {
	t.Helper()
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatal(err)
	}
	length := int(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			t.Fatal(err)
		}
		length = int(binary.BigEndian.Uint16(extended))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
