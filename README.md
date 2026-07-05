# Identity Risk Engine

![CI](https://github.com/mayanksekhar/identity-risk-engine/actions/workflows/ci.yml/badge.svg)

> Part of the [Thinkwerke](https://docs.thinkwerke.com) DevSecOps portfolio — Project 3 of the
> Detect → Audit → Prevent arc. This is the real application deployed by the
> SBOM-gated, multi-platform CI/CD pipeline; the pipeline's job is to detect,
> attest, and gate the supply chain that ships this binary to EKS and ECS.

An open-source identity risk scoring engine for human users, non-human
identities (service accounts, API keys, GitHub Apps), and autonomous AI
agents — scored against the **OWASP Non-Human Identities Top 10 (2025)**
and connected to **real GitHub organization data**, not just synthetic
demos.

## What makes this different from a portfolio demo

Most identity-risk tooling you'll find in a job-search portfolio scores
fake seed data forever. This one doesn't:

```bash
GITHUB_ORGS=your-org GITHUB_TOKEN=ghp_... go run ./cmd/server
```

Point it at a real GitHub organization and it fetches actual members, fine-
grained personal access tokens, and installed GitHub Apps via the GitHub
API, normalizes them into a common identity model, and scores every one of
them with the same deterministic rule engine — no LLM involved in the score
itself, only in the optional plain-English narration of *why* a score
landed where it did.

Every score is also mapped to the specific **OWASP NHI Top 10 (2025)**
category it violates — `NHI5:2025 Overprivileged NHI`, `NHI7:2025 Long-
Lived Secrets`, and so on — surfaced directly in the API response and as
badges on the dashboard.

Two agentic-autonomy risk factors had no clean home in that mapping — the
2025 standard predates autonomous AI agents as a named identity class — so
they're deliberately **not** force-fit into a category that wouldn't
accurately describe them. Instead they're mapped separately to the
**OWASP Top 10 for Agentic Applications (2026)** (`ASI03:2026 Identity &
Privilege Abuse`, `ASI09:2026 Human-Agent Trust Exploitation`), which does
cover this territory. Both mappings live in the API response
(`owasp_nhi_risks` and `owasp_agentic_risks`) and on the dashboard side by
side, rather than being blurred into one list.

This gap and its resolution are discussed with the OWASP NHI Top 10 project
maintainers in
[issue #43](https://github.com/OWASP/www-project-non-human-identities-top-10/issues/43).

## Why this design

- **Rule engine computes the score. The LLM only narrates it.** Every point
  added to a risk score is traceable to a named factor (`internal/risk`).
  This keeps the number audit-defensible — you can hand a regulator the
  factor list, not a black-box LLM judgment.
- **Connectors are pluggable, not hardcoded.** `internal/connectors`
  defines one interface; GitHub is the first implementation, AWS IAM and
  Kubernetes RBAC are next (see `PRODUCT-ROADMAP.md`). The scoring engine
  and dashboard never know or care which connector supplied an identity.
- **Unknown data is never assumed safe.** When a GitHub token can't see a
  member's 2FA status (scanning an org you don't administer, for
  instance), the identity is scored with the weakest auth posture and
  tagged `auth-status-unknown` — never silently assumed secure. The same
  principle applies to type-aware staleness thresholds: an AI agent idle 3
  days is a stronger signal than a human idle 3 days, because the expected
  behavior of each identity type is genuinely different.
- **Graceful LLM degradation.** If Ollama is unreachable or
  `LLM_EXPLAIN_ENABLED=false`, the service falls back to a deterministic
  templated explanation rather than failing the request.

## Run it against your own GitHub org

```bash
go mod tidy
GITHUB_ORGS=your-org-name GITHUB_TOKEN=ghp_your_token make run
```

See [`docs/github-connector-setup.md`](docs/github-connector-setup.md) for
required token scopes and what data is and isn't available depending on
whether you administer the target org.

## Run it with synthetic demo data

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

Requires a local Ollama instance:

```bash
ollama serve &
ollama pull llama3.2

LLM_EXPLAIN_ENABLED=true \
OLLAMA_BASE_URL=http://localhost:11434 \
OLLAMA_MODEL=llama3.2 \
make run
```

Then `GET /api/v1/identities/{id}` includes a populated `explanation` field
alongside the deterministic `owasp_nhi_risks` array.

## Test

```bash
make test          # 30 tests: rule engine, NHI + Agentic OWASP mappings, GitHub connector transforms
make test-cover
```

## Build & run in Docker

```bash
make docker-build
make docker-run
```

The runtime image is `distroless/static:nonroot` — no shell, no package
manager, minimal attack surface.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `GITHUB_ORGS` | (unset) | Comma-separated GitHub org logins — enables the real-data connector |
| `GITHUB_TOKEN` | (unset) | Required if `GITHUB_ORGS` is set |
| `LLM_EXPLAIN_ENABLED` | `false` | Enable Ollama-backed explanations |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama API base URL |
| `OLLAMA_MODEL` | `llama3.2` | Model used for explanation generation |
| `LLM_TIMEOUT` | `5s` | Per-request LLM timeout before fallback |

## Project structure

```
cmd/server/                  # main entrypoint, env config, graceful shutdown
internal/identity/            # core domain model
internal/risk/                  # deterministic rule-based scoring engine + NHI 2025 + Agentic 2026 OWASP mappings
internal/llm/                     # Ollama explanation client with fallback
internal/connectors/                # shared Connector interface
internal/connectors/github/           # real GitHub org data connector
internal/store/                         # in-memory store + synthetic seed data (fallback)
internal/scoring/                         # orchestration: connector/store + risk + llm -> API shape
internal/handlers/                          # HTTP routes: JSON API + dashboard
web/templates/                                # server-rendered dashboard HTML (with OWASP badges)
web/static/                                     # dashboard CSS (Thinkwerke dark/amber theme)
deploy/terraform/                                 # EKS + ECS + weighted ALB target groups
deploy/k8s/                                         # Kubernetes manifests + Kyverno signature policy
docs/                                                 # RFC, threat model, GitHub connector setup guide
```

## CI/CD

The GitHub Actions pipeline consumes a shared, versioned, reusable template
(`mayanksekhar/thinkwerke-sbom-pipeline@v1`) rather than maintaining inline
YAML — the same template any future Thinkwerke service can adopt in about
15 lines. It generates an SBOM at multiple build stages, gates the build on
Grype + Trivy findings (two independent scanners, different vulnerability
databases), signs the image and SBOM attestation with Cosign via keyless
GitHub OIDC (no static signing key), and produces SLSA build provenance.
See [`docs/rfc-sbom-gated-pipeline.md`](docs/rfc-sbom-gated-pipeline.md) and
[`docs/threat-model.md`](docs/threat-model.md) for the design rationale and
an explicit accounting of what's mitigated versus still open.

## What's next

See [`PRODUCT-ROADMAP.md`](PRODUCT-ROADMAP.md) for the full plan — AWS IAM
and Kubernetes RBAC connectors, remediation actions (PAT revocation, key
rotation), Postgres-backed persistence with score-history and drift
detection, and the path toward OWASP community recognition.
