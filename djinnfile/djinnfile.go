// Package djinnfile parses Djinnfile configuration into WorkPlans.
// MVP uses JSON format; YAML support deferred until an external
// dependency is acceptable.
package djinnfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Sentinel errors for Djinnfile parsing.
var (
	ErrNoStages   = errors.New("djinnfile: at least one stage is required")
	ErrNoStageName = errors.New("djinnfile: stage name is required")
)

// Default budget values when not specified.
const (
	DefaultTimeBudgetEco = 30 * time.Minute
	DefaultTimeBudgetSys = 15 * time.Minute
	DefaultTimeBudgetCom = 10 * time.Minute
	DefaultTimeBudgetMod = 5 * time.Minute
	DefaultModel         = "claude-sonnet-4-6"
)

// Djinnfile represents the parsed configuration.
type Djinnfile struct {
	Version string        `json:"version"`
	Name    string        `json:"name"`
	Stages  []StageConfig `json:"stages"`
	Driver  DriverConfig  `json:"driver"`
}

// StageConfig represents a single stage in the Djinnfile.
type StageConfig struct {
	Name        string        `json:"name"`
	Tier        string        `json:"tier"`   // "eco", "sys", "com", "mod"
	Scope       string        `json:"scope"`  // scope name
	Prompt      string        `json:"prompt"`
	TimeBudget  string        `json:"time_budget,omitempty"`  // Go duration string
	TokenBudget int           `json:"token_budget,omitempty"`
	Gate        GateConfig    `json:"gate"`

	parsedTimeBudget time.Duration
}

// DriverConfig holds LLM driver configuration.
type DriverConfig struct {
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// GateConfig holds gate validation configuration.
type GateConfig struct {
	Name     string             `json:"name"`
	Severity string             `json:"severity"` // "warning" or "blocking"
}

// Parse reads a Djinnfile from a reader and validates it.
func Parse(r io.Reader) (*Djinnfile, error) {
	var df Djinnfile
	if err := json.NewDecoder(r).Decode(&df); err != nil {
		return nil, fmt.Errorf("djinnfile: parse error: %w", err)
	}
	if err := df.validate(); err != nil {
		return nil, err
	}
	df.applyDefaults()
	return &df, nil
}

// ParseBytes parses a Djinnfile from a byte slice.
func ParseBytes(data []byte) (*Djinnfile, error) {
	var df Djinnfile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, fmt.Errorf("djinnfile: parse error: %w", err)
	}
	if err := df.validate(); err != nil {
		return nil, err
	}
	df.applyDefaults()
	return &df, nil
}

func (df *Djinnfile) validate() error {
	if len(df.Stages) == 0 {
		return ErrNoStages
	}
	for i := range df.Stages {
		if df.Stages[i].Name == "" {
			return fmt.Errorf("%w at index %d", ErrNoStageName, i)
		}
		if df.Stages[i].TimeBudget != "" {
			d, err := time.ParseDuration(df.Stages[i].TimeBudget)
			if err != nil {
				return fmt.Errorf("djinnfile: invalid time_budget %q for stage %q: %w",
					df.Stages[i].TimeBudget, df.Stages[i].Name, err)
			}
			df.Stages[i].parsedTimeBudget = d
		}
	}
	return nil
}

func (df *Djinnfile) applyDefaults() {
	if df.Version == "" {
		df.Version = "1"
	}
	if df.Driver.Model == "" {
		df.Driver.Model = DefaultModel
	}
	for i := range df.Stages {
		if df.Stages[i].Tier == "" {
			df.Stages[i].Tier = "mod"
		}
		if df.Stages[i].Gate.Severity == "" {
			df.Stages[i].Gate.Severity = "blocking"
		}
		if df.Stages[i].Gate.Name == "" {
			df.Stages[i].Gate.Name = df.Stages[i].Name + "-gate"
		}
		if df.Stages[i].parsedTimeBudget == 0 {
			df.Stages[i].parsedTimeBudget = defaultTimeBudgetForTier(df.Stages[i].Tier)
		}
	}
}

func defaultTimeBudgetForTier(t string) time.Duration {
	switch t {
	case "eco":
		return DefaultTimeBudgetEco
	case "sys":
		return DefaultTimeBudgetSys
	case "com":
		return DefaultTimeBudgetCom
	default:
		return DefaultTimeBudgetMod
	}
}
