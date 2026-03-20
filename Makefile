.PHONY: build test lint vet coverage clean

build:
	go build ./...

test:
	go test ./... -race -count=1

lint:
	go vet ./...

vet: lint

coverage:
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f coverage.out coverage.html
	rm -f cmd/djinn/djinn
