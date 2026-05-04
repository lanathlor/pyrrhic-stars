// Package bt provides a pure behavior tree executor with no game-specific
// dependencies. Trees are built from composite nodes (Selector, Sequence),
// decorators (Inverter), and leaf nodes (Action, Condition).
package bt

// Result is the outcome of a single node tick.
type Result int

const (
	Success Result = iota
	Failure
	Running
)

// String returns a human-readable name for the result.
func (r Result) String() string {
	switch r {
	case Success:
		return "success"
	case Failure:
		return "failure"
	case Running:
		return "running"
	default:
		return "unknown"
	}
}

// Node is a single element in a behavior tree.
type Node interface {
	Tick(ctx any) Result
}
