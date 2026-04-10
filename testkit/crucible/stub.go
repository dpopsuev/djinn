package crucible

import (
	"context"
	"sync"
	"time"
)

// StubReferee implements Referee for testing. Configurable pass/fail.
type StubReferee struct {
	mu     sync.Mutex
	result CheckResult
	err    error
	Checks int
}

var _ Referee = (*StubReferee)(nil)

// NewStubReferee creates a referee that returns the configured result.
func NewStubReferee(result CheckResult) *StubReferee {
	return &StubReferee{result: result}
}

func (r *StubReferee) Check(_ context.Context, _, _ string) (CheckResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Checks++
	return r.result, r.err
}

// SetResult changes the configured result.
func (r *StubReferee) SetResult(result CheckResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.result = result
}

// SetError configures the referee to return an error.
func (r *StubReferee) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// StubScenario implements Scenario for testing.
type StubScenario struct {
	id      string
	spec    string
	timeout time.Duration
	budget  Budget
}

var _ Scenario = (*StubScenario)(nil)

func NewStubScenario(id, spec string) *StubScenario {
	return &StubScenario{
		id:      id,
		spec:    spec,
		timeout: 10 * time.Minute,
		budget:  Budget{MaxTokens: 100000, MaxCost: 1.0, MaxTime: 10 * time.Minute},
	}
}

func (s *StubScenario) ID() string             { return s.id }
func (s *StubScenario) Spec() string           { return s.spec }
func (s *StubScenario) Timeout() time.Duration { return s.timeout }
func (s *StubScenario) Budget() Budget         { return s.budget }

// MockOperator feeds canned prompts sequentially. Implements Operator.
type MockOperator struct {
	mu      sync.Mutex
	prompts []string
	current int
	Calls   int
}

var _ Operator = (*MockOperator)(nil)

// NewMockOperator creates an operator with canned prompts.
func NewMockOperator(prompts ...string) *MockOperator {
	return &MockOperator{prompts: prompts}
}

func (o *MockOperator) Perform(_ context.Context, _ string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Calls++
	if o.current >= len(o.prompts) {
		return "", nil // exhausted
	}
	prompt := o.prompts[o.current]
	o.current++
	return prompt, nil
}
