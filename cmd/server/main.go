// Command server is the entrypoint for the Identity Risk Engine. It wires
// together the in-memory store, the deterministic risk scoring engine, and
// the optional Ollama-backed explanation layer, then serves both a JSON API
// and a server-rendered dashboard.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/handlers"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/llm"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/risk"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/scoring"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/store"
)

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=$(git rev-parse --short HEAD)"
var version = "dev"

func main() {
	port := getEnv("PORT", "8080")
	llmEnabled := getEnvBool("LLM_EXPLAIN_ENABLED", false)
	llmBaseURL := getEnv("OLLAMA_BASE_URL", "http://localhost:11434")
	llmModel := getEnv("OLLAMA_MODEL", "llama3.2")
	llmTimeout := getEnvDuration("LLM_TIMEOUT", 5*time.Second)

	memStore := store.NewMemoryStore()
	riskEngine := risk.NewEngine()
	llmClient := llm.NewClient(llm.Config{
		BaseURL: llmBaseURL,
		Model:   llmModel,
		Timeout: llmTimeout,
		Enabled: llmEnabled,
	})
	svc := scoring.NewService(memStore, riskEngine, llmClient)

	api, err := handlers.NewAPI(svc, "web/templates/*.html", version)
	if err != nil {
		log.Fatalf("failed to initialize handlers: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	api.Routes(r)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("identity-risk-engine version=%s listening on :%s (llm_enabled=%v)", version, port, llmEnabled)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGTERM (important for EKS/ECS rolling deploys —
	// both orchestrators send SIGTERM and expect a clean drain before SIGKILL).
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutdown signal received, draining connections...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("shutdown complete")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
