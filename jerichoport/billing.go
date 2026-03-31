package jerichoport

import "github.com/dpopsuev/jericho/billing"

// Type aliases — definitions live in bugle/billing.
type (
	Tracker         = billing.Tracker
	InMemoryTracker = billing.InMemoryTracker
	TokenRecord     = billing.TokenRecord
	TokenSummary    = billing.TokenSummary
	CostBill        = billing.CostBill
)

// Constructors.
var (
	NewTracker     = billing.NewTracker
	BuildCostBill  = billing.BuildCostBill
	FormatCostBill = billing.FormatCostBill
)
