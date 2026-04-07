package uniform

import "testing"

func TestParseOperation(t *testing.T) {
	tests := []struct {
		input string
		want  Operation
		ok    bool
	}{
		{"ask", OpAsk, true},
		{"plan", OpPlan, true},
		{"agent", OpAgent, true},
		{"bogus", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseOperation(tt.input)
			if ok != tt.ok {
				t.Fatalf("ParseOperation(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("ParseOperation(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOperationString(t *testing.T) {
	tests := []struct {
		op   Operation
		want string
	}{
		{OpAsk, "ask"},
		{OpPlan, "plan"},
		{OpAgent, "agent"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Fatalf("Operation.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOperationIsInterrupting(t *testing.T) {
	tests := []struct {
		op   Operation
		want bool
	}{
		{OpAsk, false},
		{OpPlan, false},
		{OpAgent, true},
	}
	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			if got := tt.op.IsInterrupting(); got != tt.want {
				t.Fatalf("%s.IsInterrupting() = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestOperationIsCascading(t *testing.T) {
	tests := []struct {
		op   Operation
		want bool
	}{
		{OpAsk, false},
		{OpPlan, true},
		{OpAgent, false},
	}
	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			if got := tt.op.IsCascading(); got != tt.want {
				t.Fatalf("%s.IsCascading() = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestOperationNext(t *testing.T) {
	tests := []struct {
		op   Operation
		want Operation
	}{
		{OpAsk, OpPlan},
		{OpPlan, OpAgent},
		{OpAgent, OpAsk},
	}
	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			if got := tt.op.Next(); got != tt.want {
				t.Fatalf("%s.Next() = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestDefaultOperation(t *testing.T) {
	if got := DefaultOperation(); got != OpAgent {
		t.Fatalf("DefaultOperation() = %q, want %q", got, OpAgent)
	}
}
