// Package taskforce classifies task complexity and composes purpose-built
// agent formations. The Broker uses the Classifier to determine the
// complexity band, then the Composer selects and instantiates the
// appropriate formation template.
package taskforce

// ComplexityBand classifies task complexity using the Cynefin framework.
type ComplexityBand int

const (
	Clear       ComplexityBand = iota // Trivial: 1 file, 0 cross-pkg imports
	Complicated                      // Bounded: 1 package
	Complex                          // Multi-package or multi-repo
	Chaotic                          // Emergency: inverted cascade
)

func (b ComplexityBand) String() string {
	switch b {
	case Clear:
		return "clear"
	case Complicated:
		return "complicated"
	case Complex:
		return "complex"
	case Chaotic:
		return "chaotic"
	default:
		return "unknown"
	}
}
