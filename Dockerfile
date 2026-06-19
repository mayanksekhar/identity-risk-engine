# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.26.4-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/identity-risk-engine ./cmd/server

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /out/identity-risk-engine /app/identity-risk-engine
COPY --from=builder /src/web /app/web

USER nonroot:nonroot

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["/app/identity-risk-engine"]
