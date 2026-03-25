package bugleport

import "github.com/dpopsuev/bugle/signal"

// Type aliases — definitions live in bugle/signal.
type (
	Signal   = signal.Signal
	Bus      = signal.Bus
	MemBus   = signal.MemBus
	DurableBus = signal.DurableBus
)

// Performative constants.
const (
	Inform    = signal.Inform
	Request   = signal.Request
	Confirm   = signal.Confirm
	Refuse    = signal.Refuse
	Handoff   = signal.Handoff
	Directive = signal.Directive
)

// Event constants.
const (
	EventWorkerStarted  = signal.EventWorkerStarted
	EventWorkerStopped  = signal.EventWorkerStopped
	EventWorkerDone     = signal.EventWorkerDone
	EventWorkerError    = signal.EventWorkerError
	EventShouldStop     = signal.EventShouldStop
	EventBudgetUpdate   = signal.EventBudgetUpdate
	EventDispatchRouted = signal.EventDispatchRouted
)

// Meta key constants.
const (
	MetaKeyWorkerID = signal.MetaKeyWorkerID
	MetaKeyError    = signal.MetaKeyError
)

// Constructors.
var (
	NewMemBus    = signal.NewMemBus
	NewDurableBus = signal.NewDurableBus
)
