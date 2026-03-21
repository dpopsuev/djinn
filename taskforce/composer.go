package taskforce

import (
	"errors"
	"fmt"
	"time"

	"github.com/dpopsuev/djinn/composition"
	"github.com/dpopsuev/djinn/tier"
)

// Sentinel errors for composition.
var (
	ErrNoScopes = errors.New("at least one scope is required")
)

// Default formation templates per complexity band.
var defaultTemplates = map[ComplexityBand]string{
	Clear:       "solo",
	Complicated: "duo",
	Complex:     "squad",
	Chaotic:     "duo",
}

// Composer classifies intent complexity and assembles a TaskForce.
type Composer struct {
	classifier Classifier
	overrides  map[ComplexityBand]composition.Formation
}

// NewComposer creates a composer with the given classifier.
func NewComposer(classifier Classifier) *Composer {
	return &Composer{
		classifier: classifier,
		overrides:  make(map[ComplexityBand]composition.Formation),
	}
}

// WithOverride sets a custom formation for a specific complexity band.
func (c *Composer) WithOverride(band ComplexityBand, formation composition.Formation) {
	c.overrides[band] = formation
}

// Compose classifies the task and assembles an immutable TaskForce.
func (c *Composer) Compose(id string, scopes []tier.Scope, fileCount, repoCount int, budget composition.Budget) (TaskForce, error) {
	if len(scopes) == 0 {
		return TaskForce{}, ErrNoScopes
	}

	band := c.classifier.Classify(scopes, fileCount, repoCount)

	tmpl, err := c.templateForBand(band)
	if err != nil {
		return TaskForce{}, fmt.Errorf("compose %q: %w", id, err)
	}

	scopeName := scopes[0].Name
	formation, err := composition.Instantiate(tmpl, scopeName, budget)
	if err != nil {
		return TaskForce{}, fmt.Errorf("compose %q: %w", id, err)
	}

	return TaskForce{
		ID:             id,
		Band:           band,
		Formation:      formation,
		Budget:         budget,
		WatchdogConfig: DefaultWatchdogAssignment(band),
		CreatedAt:      time.Now(),
	}, nil
}

func (c *Composer) templateForBand(band ComplexityBand) (composition.Formation, error) {
	if override, ok := c.overrides[band]; ok {
		return override, nil
	}
	name, ok := defaultTemplates[band]
	if !ok {
		return composition.Formation{}, fmt.Errorf("no template for band %v", band)
	}
	return composition.TemplateByName(name)
}
