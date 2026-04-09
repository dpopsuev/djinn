package arena

import "time"

// HTTPServiceScenario returns the "Easy" arena scenario:
// build an HTTP server with a /health endpoint.
func HTTPServiceScenario() Scenario {
	return NewStubScenario(
		"http-service",
		"Build a Go HTTP server with a /health endpoint that returns "+
			"JSON {\"status\": \"ok\"} on GET requests. "+
			"The server should read the PORT environment variable for the listen port, "+
			"defaulting to 8080 if not set. "+
			"Use only stdlib (net/http, os). Single main.go file.",
	)
}

// HTTPServiceFixture is a known-good HTTP server for testing the referee.
// Reads PORT env var (set by referee for dynamic port allocation).
// Falls back to 8080 if PORT is not set.
const HTTPServiceFixture = `package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ` + "`" + `{"status":"ok"}` + "`" + `)
	})
	http.ListenAndServe(":"+port, mux)
}
`

func init() {
	// Override default timeout/budget for HTTP scenario.
	_ = time.Minute // ensure import
}
