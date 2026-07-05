// Command server is the entrypoint for the Identity Risk Engine. It wires
// together an identity data source (synthetic seed data, or a real GitHub
// connector when GITHUB_ORGS is set), the deterministic risk scoring
// engine, and the optional Ollama-backed explanation layer, then serves
// both a JSON API and a server-rendered dashboard.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/connectors/github"
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

	idStore := buildIdentityStore()

	riskEngine := risk.NewEngine()
	llmClient := llm.NewClient(llm.Config{
		BaseURL: llmBaseURL,
		Model:   llmModel,
		Timeout: llmTimeout,
		Enabled: llmEnabled,
	})
	svc := scoring.NewService(idStore, riskEngine, llmClient)

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

	// Graceful shutdown on SIGTERM (important for EKS/ECS rolling deploys -
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

// buildIdentityStore selects the identity data source. Setting GITHUB_ORGS
// switches from the synthetic in-memory seed data to a live GitHub
// connector scanning the given comma-separated organization logins. This
// is the only place that decision is made - everything downstream
// (scoring, API, dashboard) works identically regardless of which store is
// selected, since both satisfy scoring.IdentityStore.
func buildIdentityStore() scoring.IdentityStore {
	orgsEnv := getEnv("GITHUB_ORGS", "")
	if orgsEnv == "" {
		log.Println("GITHUB_ORGS not set - using synthetic in-memory seed data")
		return store.NewMemoryStore()
	}

	token := getEnv("GITHUB_TOKEN", "")
	if token == "" {
		log.Fatal("GITHUB_ORGS is set but GITHUB_TOKEN is empty - a token is required to query the GitHub API")
	}

	orgs := strings.Split(orgsEnv, ",")
	for i := range orgs {
		orgs[i] = strings.TrimSpace(orgs[i])
	}

	log.Printf("using GitHub connector for orgs: %v (data completeness depends on token permissions per org)", orgs)
	return github.New(token, orgs)
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
