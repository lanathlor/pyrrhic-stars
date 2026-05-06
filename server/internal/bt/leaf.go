package bt

// Action wraps a function as a leaf node. The function receives the tree
// context and returns Success, Failure, or Running.
type Action struct {
	Fn func(ctx any) Result
}

func NewAction(fn func(any) Result) *Action {
	return &Action{Fn: fn}
}

func (a *Action) Tick(ctx any) Result {
	return a.Fn(ctx)
}

// Condition wraps a boolean predicate as a leaf node. Returns Success when
// the predicate is true, Failure when false. Conditions must not have side effects.
type Condition struct {
	Fn func(ctx any) bool
}

func NewCondition(fn func(any) bool) *Condition {
	return &Condition{Fn: fn}
}

func (c *Condition) Tick(ctx any) Result {
	if c.Fn(ctx) {
		return Success
	}
	return Failure
}

// NamedNode wraps a Node and associates a human-readable label with it.
// Used so that instrumentation can report meaningful names for leaf nodes.
type NamedNode struct {
	Inner Node
	Label string
}

func Named(label string, inner Node) *NamedNode {
	return &NamedNode{Inner: inner, Label: label}
}

func (n *NamedNode) Tick(ctx any) Result { return n.Inner.Tick(ctx) }

// NodeName returns the label. Used by instrumentation to display readable names.
func (n *NamedNode) NodeName() string { return n.Label }
