.PHONY: build install test test-accept lint vet circuit coverage clean doctor preflight smoke-claude smoke-vertex smoke-all

build:
	go build ./cmd/djinn/

install:
	go install ./cmd/djinn/

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

smoke-all: smoke-claude smoke-vertex

preflight: lint vet test install
	djinn doctor

coverage:
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f coverage.out coverage.html
	go clean -cache -testcache
