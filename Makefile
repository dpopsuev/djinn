VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/dpopsuev/djinn/app.Version=$(VERSION)"

.PHONY: build install test test-accept lint vet circuit coverage clean doctor preflight smoke-claude smoke-vertex smoke-gemini smoke-codex smoke-cursor smoke-agents smoke-all

build:
	go build $(LDFLAGS) ./cmd/djinn/

install:
	go install $(LDFLAGS) ./cmd/djinn/

test:
	go test ./... -race -count=1 -timeout=60s

test-accept:
	go test ./acceptance/ -race -v -timeout=60s

lint:
	golangci-lint run ./...

vet:
	go vet ./...

circuit: build lint test test-accept
	@echo "Circuit complete — all gates passed"

smoke-claude:
	go test ./driver/claude/ -tags=e2e -run TestSmoke_ClaudeDirect -race -v -timeout=120s

smoke-vertex:
	go test ./driver/claude/ -tags=e2e -run TestSmoke_Vertex -race -v -timeout=120s

smoke-gemini:
	go test ./driver/gemini/ -tags=e2e -run TestSmoke_Gemini -race -v -timeout=60s

smoke-codex:
	go test ./driver/codex/ -tags=e2e -run TestSmoke_Codex -race -v -timeout=60s

smoke-cursor:
	go test ./driver/cursor/ -tags=e2e -run TestSmoke_Cursor -race -v -timeout=60s

smoke-agents:
	go test ./acceptance/ -tags=e2e -run TestSmoke -race -v -timeout=120s

smoke-all: smoke-claude smoke-vertex smoke-gemini smoke-codex smoke-cursor

preflight: lint vet test install
	djinn doctor

coverage:
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f coverage.out coverage.html
	go clean -cache -testcache
