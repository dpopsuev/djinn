package claude

import (
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/dpopsuev/djinn/driver"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxRetries int           // max retry attempts (default 3)
	BaseDelay  time.Duration // initial delay (default 1s, exponential: 1s, 2s, 4s)
	MaxDelay   time.Duration // delay cap (default 30s)
}

// DefaultRetryConfig returns production defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// retryTransport wraps an http.RoundTripper with exponential backoff retry
// for transient HTTP errors (5xx, 429) and network errors.
type retryTransport struct {
	inner  http.RoundTripper
	config RetryConfig
	log    *slog.Logger
	sleep  func(time.Duration) // injectable for testing
}

func newRetryTransport(inner http.RoundTripper, config RetryConfig, log *slog.Logger) *retryTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &retryTransport{
		inner:  inner,
		config: config,
		log:    log,
		sleep:  time.Sleep,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := t.backoff(attempt)
			t.log.Warn("retrying request",
				"attempt", attempt,
				"delay", delay,
				"url", req.URL.String(),
			)
			t.sleep(delay)

			// Re-read body for retry.
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}

		resp, err := t.inner.RoundTrip(req)
		if err != nil {
			// Network error — retryable.
			lastErr = err
			lastResp = nil
			continue
		}

		if !driver.ClassifyRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Retryable status — close body before retry, keep last for return.
		lastResp = resp
		lastErr = nil
		if attempt < t.config.MaxRetries {
			resp.Body.Close()
		}
	}

	if lastResp != nil {
		return lastResp, lastErr
	}
	return nil, lastErr
}

func (t *retryTransport) backoff(attempt int) time.Duration {
	delay := time.Duration(float64(t.config.BaseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > t.config.MaxDelay {
		delay = t.config.MaxDelay
	}
	return delay
}
