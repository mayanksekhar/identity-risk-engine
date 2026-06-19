# Identity Risk Engine

> Part of the [Thinkwerke](https://docs.thinkwerke.com) DevSecOps portfolio — Project 3 of the
> Detect → Audit → Prevent arc. This is the real application deployed by the
> SBOM-gated, multi-platform CI/CD pipeline; the pipeline's job is to detect,
> attest, and gate the supply chain that ships this binary to EKS and ECS.

A consolidated identity risk management platform for enterprise identities —
human users, non-human identities (service accounts, API keys), and AI
agents. Combines a deterministic, fully explainable rule-based scoring
engine with an optional LLM narrative layer (local Ollama) that explains
*why* an identity scored the way it did, without ever letting the LLM
compute the score itself.

## Why this design

- **Rule engine computes the score. The LLM only narrates it.** Every point
  added to a risk score is traceable to a named factor (`internal/risk`).
  This keeps the number audit-defensible — you can hand a regulator the
  factor list, not a black-box LLM judgment — while still delivering the
  "agentic" explanation layer the platform name promises.
- **Graceful LLM degradation.** If Ollama is unreachable or `LLM_EXPLAIN_ENABLED=false`,
  the service falls back to a deterministic templated explanation rather
  than failing the request. Risk visibility must never depend on LLM uptime.
- **Type-aware staleness thresholds.** A human inactive for 5 days is
  normal. An AI agent inactive for 5 days is a strong signal of an orphaned
  credential — `internal/risk/engine.go` encodes different staleness
  thresholds per identity type rather than one global cutoff.
- **Zero external dependencies in the demo build.** In-memory store, no DB,
  no required Redis. This matters for the EKS/ECS split demo: identical
  behavior regardless of which backend serves a given request.

## Run locally

```bash
go mod tidy
make run
# or: go run ./cmd/server
```

Visit `http://localhost:8080` for the dashboard, or:

```bash
curl localhost:8080/api/v1/summary
curl localhost:8080/api/v1/identities
curl localhost:8080/api/v1/identities/agt-3002
curl localhost:8080/healthz
```

## Enable LLM explanations (optional)

Requires a local Ollama instance (already part of your son-of-anton toolchain
from Project 2):

```bash
ollama serve &
ollama pull llama3.2

LLM_EXPLAIN_ENABLED=true \
OLLAMA_BASE_URL=http://localhost:11434 \
OLLAMA_MODEL=llama3.2 \
make run
```

Then `GET /api/v1/identities/{id}` will include a populated `explanation`
field instead of the rule-engine fallback summary.

## Test

```bash
make test
make test-cover
```

## Build & run in Docker

```bash
make docker-build
make docker-run
```

The runtime image is `distroless/static:nonroot` — no shell, no package
manager, minimal attack surface. This is intentional: it also means a shell
spawned inside this container (e.g. via a compromised dependency) is a much
stronger Falco signal, consistent with the runtime-security narrative from
Project 1.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `LLM_EXPLAIN_ENABLED` | `false` | Enable Ollama-backed explanations |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama API base URL |
| `OLLAMA_MODEL` | `llama3.2` | Model used for explanation generation |
| `LLM_TIMEOUT` | `5s` | Per-request LLM timeout before fallback |

## Project structure

```
cmd/server/          # main entrypoint, env config, graceful shutdown
internal/identity/    # core domain model
internal/risk/         # deterministic rule-based scoring engine (+ tests)
internal/llm/           # Ollama explanation client with fallback
internal/store/          # in-memory store + synthetic seed data
internal/scoring/         # orchestration: store + risk + llm -> API shape
internal/handlers/         # HTTP routes: JSON API + dashboard
web/templates/               # server-rendered dashboard HTML
web/static/                   # dashboard CSS (Thinkwerke dark/amber theme)
deploy/terraform/              # EKS + ECS + weighted ALB target groups
deploy/k8s/                     # Kubernetes manifests for EKS deployment
deploy/ecs/                      # ECS task definition / service
```

## Next: CI/CD and the 80/20 EKS/ECS split

The GitLab CI pipeline (`.gitlab-ci.yml` at repo root) generates the SBOM at
five stages (commit, build, registry, and scheduled re-scan), gates the
build on Syft + Grype + Trivy findings, signs the image and SBOM attestation
with Cosign, then deploys behind a single ALB with weighted target groups:
80% of traffic to EKS pods, 20% to ECS Fargate tasks. See `deploy/terraform/`
for the infrastructure definition.
