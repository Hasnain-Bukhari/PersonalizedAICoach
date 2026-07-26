package main

import (
	"github.com/personalized-ai-coach/backend/internal/adapters/identity"
	"github.com/personalized-ai-coach/backend/internal/adapters/llm"
	"github.com/personalized-ai-coach/backend/internal/adapters/memory"
	"github.com/personalized-ai-coach/backend/internal/api"
	"github.com/personalized-ai-coach/backend/internal/application"
	"github.com/personalized-ai-coach/backend/internal/ports"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	store := memory.New()
	models := map[string]string{"planning": env("LLM_PLANNING_MODEL", "coach-default"), "teaching": env("LLM_TEACHING_MODEL", "coach-default"), "assessment": env("LLM_ASSESSMENT_MODEL", "coach-default"), "interview": env("LLM_INTERVIEW_MODEL", "coach-default")}
	var model ports.LLM = llm.Fake{}
	if base := os.Getenv("LLM_BASE_URL"); base != "" {
		model = llm.New(base, os.Getenv("LLM_API_KEY"), models)
	}
	coach := application.New(store, model)
	authMode := "auth0"
	appEnv := env("APP_ENV", "production")
	if appEnv == "local" || appEnv == "development" {
		authMode = "dev"
	}
	issuer, audience := os.Getenv("AUTH0_ISSUER"), os.Getenv("AUTH0_AUDIENCE")
	if authMode == "auth0" && (issuer == "" || audience == "") {
		log.Error("production authentication configuration is incomplete", "required", "AUTH0_ISSUER, AUTH0_AUDIENCE")
		os.Exit(1)
	}
	verifier := identity.New(env("AUTH0_ISSUER", "https://local.invalid/"), env("AUTH0_AUDIENCE", "ai-learning-coach"), authMode)
	server := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: api.New(coach, store, verifier, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	log.Info("api listening", "address", server.Addr, "auth_mode", verifier.Mode)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
