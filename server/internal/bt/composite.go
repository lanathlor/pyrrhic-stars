package bt

// Selector tries children left-to-right. Returns Success on the first child
// that succeeds, Failure if all children fail. When a child returns Running,
// the selector remembers its index and resumes from that child on the next tick.
type Selector struct {
	Children   []Node
	runningIdx int
}

func NewSelector(children ...Node) *Selector {
	return &Selector{Children: children, runningIdx: -1}
}

func (s *Selector) Tick(ctx any) Result {
	start := max(0, s.runningIdx)
	for i := start; i < len(s.Children); i++ {
		r := s.Children[i].Tick(ctx)
		switch r {
		case Success:
			s.runningIdx = -1
			return Success
		case Running:
			s.runningIdx = i
			return Running
		}
		// Failure: try next child
	}
	s.runningIdx = -1
	return Failure
}

// ReactiveSelector evaluates children from child 0 every tick, regardless of
// which child was previously Running. This allows higher-priority branches to
// interrupt lower-priority running behaviors. When a different child takes over,
// the previously-running child's subtree is reset.
type ReactiveSelector struct {
	Children   []Node
	runningIdx int
}

func NewReactiveSelector(children ...Node) *ReactiveSelector {
	return &ReactiveSelector{Children: children, runningIdx: -1}
}

func (s *ReactiveSelector) Tick(ctx any) Result {
	for i := range s.Children {
		r := s.Children[i].Tick(ctx)
		switch r {
		case Success:
			if s.runningIdx >= 0 && s.runningIdx != i {
				resetNode(s.Children[s.runningIdx])
			}
			s.runningIdx = -1
			return Success
		case Running:
			if s.runningIdx >= 0 && s.runningIdx != i {
				resetNode(s.Children[s.runningIdx])
			}
			s.runningIdx = i
			return Running
		}
	}
	if s.runningIdx >= 0 {
		resetNode(s.Children[s.runningIdx])
	}
	s.runningIdx = -1
	return Failure
}

// Sequence runs children left-to-right. Returns Failure on the first child
// that fails, Success if all children succeed. When a child returns Running,
// the sequence remembers its index and resumes from that child on the next tick.
type Sequence struct {
	Children   []Node
	runningIdx int
}

func NewSequence(children ...Node) *Sequence {
	return &Sequence{Children: children, runningIdx: -1}
}

func (s *Sequence) Tick(ctx any) Result {
	start := max(0, s.runningIdx)
	for i := start; i < len(s.Children); i++ {
		r := s.Children[i].Tick(ctx)
		switch r {
		case Failure:
			s.runningIdx = -1
			return Failure
		case Running:
			s.runningIdx = i
			return Running
		}
		// Success: continue to next child
	}
	s.runningIdx = -1
	return Success
}
