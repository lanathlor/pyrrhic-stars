package bt

// Tree wraps a root node and provides top-level tick and reset operations.
type Tree struct {
	Root Node
}

func NewTree(root Node) *Tree {
	return &Tree{Root: root}
}

func (t *Tree) Tick(ctx any) Result {
	return t.Root.Tick(ctx)
}

// Reset clears running state on all composites in the tree. This is useful
// when an entity needs to restart its behavior (e.g., after a phase transition
// or leash reset).
func (t *Tree) Reset() {
	resetNode(t.Root)
}

func resetNode(n Node) {
	switch v := n.(type) {
	case *Selector:
		v.runningIdx = -1
		for _, c := range v.Children {
			resetNode(c)
		}
	case *ReactiveSelector:
		v.runningIdx = -1
		for _, c := range v.Children {
			resetNode(c)
		}
	case *Sequence:
		v.runningIdx = -1
		for _, c := range v.Children {
			resetNode(c)
		}
	case *Inverter:
		resetNode(v.Child)
	case *NamedNode:
		resetNode(v.Inner)
	}
}
