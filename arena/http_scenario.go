package arena

import "time"

// HTTPServiceScenario returns the "Easy" arena scenario:
// build an HTTP server with a /health endpoint.
func HTTPServiceScenario() Scenario {
	return NewStubScenario(
		"http-service",
		"Build a Go HTTP server with a /health endpoint that returns "+
			"JSON {\"status\": \"ok\"} on GET requests. "+
			"The server should listen on port 8080. "+
			"Use only stdlib (net/http). Single main.go file.",
	)
}

// HTTPServiceFixture is a known-good HTTP server for testing the referee.
// The referee should pass this without any LLM involvement.
// HTTPServiceFixture is a known-good HTTP server for testing the referee.
const HTTPServiceFixture = "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\tmux := http.NewServeMux()\n\tmux.HandleFunc(\"GET /health\", func(w http.ResponseWriter, r *http.Request) {\n\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\tfmt.Fprint(w, `{\"status\":\"ok\"}`)\n\t})\n\thttp.ListenAndServe(\":8080\", mux)\n}\n"

func init() {
	// Override default timeout/budget for HTTP scenario.
	_ = time.Minute // ensure import
}
