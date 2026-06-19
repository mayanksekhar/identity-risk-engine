// Package handlers wires HTTP routes to the scoring service, for both the
// JSON API and the server-rendered dashboard.
package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/scoring"
)

// API holds shared dependencies for HTTP handlers.
type API struct {
	svc       *scoring.Service
	tmpl      *template.Template
	startedAt time.Time
	version   string // set at build time via -ldflags, surfaced on /healthz
}

// NewAPI constructs the handler set. tmplPath points at the directory
// containing dashboard templates (web/templates).
func NewAPI(svc *scoring.Service, tmplGlob string, version string) (*API, error) {
	tmpl, err := template.ParseGlob(tmplGlob)
	if err != nil {
		return nil, err
	}
	return &API{svc: svc, tmpl: tmpl, startedAt: time.Now(), version: version}, nil
}

// Routes registers all HTTP routes on the given chi router.
func (a *API) Routes(r chi.Router) {
	r.Get("/healthz", a.handleHealthz)
	r.Get("/readyz", a.handleReadyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/identities", a.handleListIdentities)
		r.Get("/identities/{id}", a.handleGetIdentity)
		r.Get("/summary", a.handleSummary)
	})

	r.Get("/", a.handleDashboard)
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
}

// handleHealthz reports liveness only — it must never depend on downstream
// dependencies (LLM, future DB) so the ALB target group health check can't
// be taken down by an unrelated outage.
func (a *API) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"uptime_secs":  int(time.Since(a.startedAt).Seconds()),
		"version":      a.version,
		"backend_host": hostname(),
	})
}

// handleReadyz is a placeholder for future dependency checks (DB, Redis).
// Currently identical to healthz since the demo build has no hard external
// dependency — the LLM layer degrades gracefully rather than blocking
// readiness.
func (a *API) handleReadyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (a *API) handleListIdentities(w http.ResponseWriter, r *http.Request) {
	results := a.svc.ListScored()
	writeJSON(w, http.StatusOK, results)
}

func (a *API) handleGetIdentity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, ok := a.svc.GetScored(r.Context(), id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "identity not found"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.svc.Summarize())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
