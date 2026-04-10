package budget

var (
	_ Observer   = (*StubObserver)(nil)
	_ Controller = (*StubController)(nil)
)

// StubObserver implements Observer for testing.
type StubObserver struct {
	ExceededVal bool
	UsageVal    float64
}

func (s *StubObserver) Exceeded() bool { return s.ExceededVal }
func (s *StubObserver) Usage() float64 { return s.UsageVal }

// StubController implements Controller for testing.
type StubController struct {
	ThrottleVal bool
	KillVal     bool
}

func (s *StubController) ShouldThrottle() bool { return s.ThrottleVal }
func (s *StubController) ShouldKill() bool     { return s.KillVal }
