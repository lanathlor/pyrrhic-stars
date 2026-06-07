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

// StripNames recursively removes NamedNode wrappers from the tree, reducing
// interface dispatch overhead in the hot path. Names are only needed for debug
// logging and instrumentation; production trees should be stripped after build.
func StripNames(n Node) Node {
	switch v := n.(type) {
	case *NamedNode:
		return StripNames(v.Inner)
	case *Selector:
		for i, c := range v.Children {
			v.Children[i] = StripNames(c)
		}
	case *ReactiveSelector:
		for i, c := range v.Children {
			v.Children[i] = StripNames(c)
		}
	case *Sequence:
		for i, c := range v.Children {
			v.Children[i] = StripNames(c)
		}
	case *Inverter:
		v.Child = StripNames(v.Child)
	}
	return n
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
