VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/dpopsuev/djinn/app.Version=$(VERSION)"

.PHONY: build install test test-accept fmt lint lint-new lint-tui vet circuit coverage clean doctor preflight install-hooks smoke-claude smoke-vertex smoke-gemini smoke-codex smoke-cursor smoke-agents smoke-all

build:
	go build $(LDFLAGS) ./cmd/djinn/

install:
	go install $(LDFLAGS) ./cmd/djinn/

test:
	go test ./... -race -count=1 -timeout=60s

test-accept:
	go test ./acceptance/ -race -v -timeout=60s

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

lint-new:
	golangci-lint run --new-from-rev=HEAD ./...

vet:
	go vet ./...

lint-tui:
	@echo "Checking no .Background() calls in tui/..."
	@! grep -rn '\.Background(' tui/ --include='*.go' | grep -v '_test.go' || (echo "FAIL: .Background() found in tui/" && exit 1)
	@echo "OK: no Background() calls"
	@echo "Checking no raw hex outside colors.go/theme.go..."
	@! grep -rn '"#[0-9a-fA-F]\{6\}"' tui/ --include='*.go' | grep -v '_test.go' | grep -v 'colors.go' | grep -v 'theme.go' || (echo "FAIL: raw hex found outside colors.go/theme.go" && exit 1)
	@echo "OK: no raw hex"

circuit: fmt vet build lint lint-tui test test-accept
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

preflight: fmt vet lint lint-tui test install
	djinn doctor

coverage:
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

install-hooks:
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make lint-new' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed (runs make lint-new)"

clean:
	rm -f coverage.out coverage.html
	go clean -cache -testcache
