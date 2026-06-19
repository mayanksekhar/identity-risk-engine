.PHONY: run build test test-cover lint fmt vet docker-build docker-run clean tidy

VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IMAGE   ?= identity-risk-engine:$(VERSION)

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/identity-risk-engine ./cmd/server

test:
	go test ./... -v

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	gofmt -l .
	go vet ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

docker-run: docker-build
	docker run --rm -p 8080:8080 \
		-e LLM_EXPLAIN_ENABLED=false \
		$(IMAGE)

clean:
	rm -rf bin coverage.out
