package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"io"
	"net/http"
	"strings"
	"time"
)

// Gateway targets any OpenAI-compatible /v1/chat/completions service. Model
// aliases keep application code independent from the deployed model names.
type Gateway struct {
	BaseURL, APIKey string
	Models          map[string]string
	Client          *http.Client
}

const maxModelResponseBytes = 2 << 20

func New(base, key string, models map[string]string) *Gateway {
	return &Gateway{BaseURL: strings.TrimRight(base, "/"), APIKey: key, Models: models, Client: &http.Client{Timeout: 15 * time.Second}}
}
func (g *Gateway) Complete(ctx context.Context, r ports.LLMRequest) (ports.LLMResponse, error) {
	model := r.Model
	if x := g.Models[r.Task]; x != "" {
		model = x
	}
	if model == "" {
		model = "coach-default"
	}
	body := map[string]any{"model": model, "messages": []map[string]string{{"role": "system", "content": r.System}, {"role": "user", "content": r.Prompt}}, "temperature": 0.2}
	if r.JSON {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	b, _ := json.Marshal(body)
	endpoint := g.BaseURL + "/v1/chat/completions"
	if strings.HasSuffix(g.BaseURL, "/v1") {
		endpoint = g.BaseURL + "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return ports.LLMResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if g.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.APIKey)
	}
	resp, err := g.Client.Do(req)
	if err != nil {
		return ports.LLMResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return ports.LLMResponse{}, fmt.Errorf("model gateway returned status %d", resp.StatusCode)
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModelResponseBytes+1))
	if err != nil {
		return ports.LLMResponse{}, errors.New("unable to read model response")
	}
	if len(body) > maxModelResponseBytes {
		return ports.LLMResponse{}, errors.New("model response is too large")
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return ports.LLMResponse{}, errors.New("model returned invalid JSON")
	}
	if len(out.Choices) == 0 {
		return ports.LLMResponse{}, errors.New("model returned no choices")
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return ports.LLMResponse{}, errors.New("model returned empty content")
	}
	return ports.LLMResponse{Content: content, InputTokens: out.Usage.Prompt, OutputTokens: out.Usage.Completion, Model: out.Model}, nil
}

type Fake struct{}

func (Fake) Complete(_ context.Context, r ports.LLMRequest) (ports.LLMResponse, error) {
	return ports.LLMResponse{Content: `{"ok":true}`, InputTokens: 10, OutputTokens: 5, Model: "deterministic-fake"}, nil
}
