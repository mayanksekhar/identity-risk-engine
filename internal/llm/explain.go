// Package llm provides a thin client for generating human-readable
// explanations of rule-based risk scores via a local Ollama instance.
// The LLM never computes the score itself — it only narrates a score
// that has already been deterministically calculated by package risk.
// This separation keeps the audit-relevant number explainable and
// reproducible while still giving an "agentic" narrative layer on top.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/risk"
)

// Client talks to a local Ollama server's /api/generate endpoint.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
	enabled    bool
}

// Config controls how the LLM explanation client is constructed.
type Config struct {
	BaseURL string        // e.g. http://localhost:11434
	Model   string        // e.g. "llama3.2"
	Timeout time.Duration // request timeout; explanations must not block the dashboard
	Enabled bool          // allows full disable in environments without Ollama (e.g. CI, ECS demo)
}

// NewClient builds an explanation client. If Enabled is false, all calls
// return a deterministic fallback string instead of making network calls —
// this matters for the EKS/ECS demo where Ollama may not be reachable from
// every target group.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		enabled:    cfg.Enabled,
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// Explain produces a short, human-readable narrative for a single identity's
// risk score. On any failure (timeout, connection refused, non-2xx), it
// degrades to a deterministic template-based explanation rather than
// failing the request — risk visibility must never depend on LLM uptime.
func (c *Client) Explain(ctx context.Context, id identity.Identity, score risk.Score) string {
	if !c.enabled {
		return fallbackExplanation(id, score)
	}

	prompt := buildPrompt(id, score)
	reqBody, err := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false})
	if err != nil {
		return fallbackExplanation(id, score)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return fallbackExplanation(id, score)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fallbackExplanation(id, score)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackExplanation(id, score)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackExplanation(id, score)
	}

	var out generateResponse
	if err := json.Unmarshal(body, &out); err != nil || strings.TrimSpace(out.Response) == "" {
		return fallbackExplanation(id, score)
	}

	return strings.TrimSpace(out.Response)
}

func buildPrompt(id identity.Identity, score risk.Score) string {
	var factorLines strings.Builder
	for _, f := range score.Factors {
		sign := "+"
		if f.Weight < 0 {
			sign = ""
		}
		fmt.Fprintf(&factorLines, "- %s (%s%d points)\n", f.Label, sign, f.Weight)
	}

	return fmt.Sprintf(
		"You are a security analyst assistant. Write a 2-3 sentence plain-English "+
			"explanation of why this identity has a risk score of %d/100 (%s severity). "+
			"Be specific and reference the actual factors. Do not invent facts not listed below. "+
			"Identity: %s (type: %s, privilege: %s).\nContributing factors:\n%s",
		score.Total, score.Severity, id.Name, id.Type, id.Privilege, factorLines.String(),
	)
}

// fallbackExplanation is used when the LLM is disabled or unreachable. It is
// deliberately template-based and deterministic so that the dashboard never
// shows a blank explanation field.
func fallbackExplanation(id identity.Identity, score risk.Score) string {
	if len(score.Factors) == 0 {
		return fmt.Sprintf("%s has a clean risk profile with no contributing risk factors on record.", id.Name)
	}
	top := score.Factors[0]
	return fmt.Sprintf(
		"%s scored %d/100 (%s). The largest contributing factor is: %s. "+
			"(LLM narrative unavailable; showing rule-engine summary.)",
		id.Name, score.Total, score.Severity, top.Label,
	)
}
