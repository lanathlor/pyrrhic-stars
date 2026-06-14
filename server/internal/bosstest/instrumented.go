package bosstest

import (
	"sync/atomic"

	"codex-online/server/internal/bt"
)

// Classification describes how a BT node performed during a fuzz run.
type Classification uint8

const (
	ClassDead    Classification = iota // 0 evaluations — structurally unreachable
	ClassCold                          // evaluated, 0 successes — condition never met
	ClassHot                           // >90% tick hit rate — dominant path
	ClassRare                          // <1% tick hit rate — edge case
	ClassHealthy                       // normal operation
)

func (c Classification) String() string {
	switch c {
	case ClassDead:
		return "dead"
	case ClassCold:
		return "cold"
	case ClassHot:
		return "hot"
	case ClassRare:
		return "rare"
	default:
		return "healthy"
	}
}

// InstrumentedNode wraps a bt.Node and tracks evaluation metrics.
type InstrumentedNode struct {
	inner        bt.Node
	Name         string
	evalCount    atomic.Int64
	successCount atomic.Int64
	failCount    atomic.Int64
	runningCount atomic.Int64
}

func (n *InstrumentedNode) Tick(ctx any) bt.Result {
	n.evalCount.Add(1)
	r := n.inner.Tick(ctx)
	switch r {
	case bt.Success:
		n.successCount.Add(1)
	case bt.Failure:
		n.failCount.Add(1)
	case bt.Running:
		n.runningCount.Add(1)
	}
	return r
}

// EvalCount returns how many times the node was evaluated.
func (n *InstrumentedNode) EvalCount() int64 { return n.evalCount.Load() }

// SuccessCount returns how many evaluations returned Success.
func (n *InstrumentedNode) SuccessCount() int64 { return n.successCount.Load() }

// FailCount returns how many evaluations returned Failure.
func (n *InstrumentedNode) FailCount() int64 { return n.failCount.Load() }

// RunningCount returns how many evaluations returned Running.
func (n *InstrumentedNode) RunningCount() int64 { return n.runningCount.Load() }

// Classify determines the node classification based on total ticks observed.
func (n *InstrumentedNode) Classify(totalTicks int64) Classification {
	evals := n.evalCount.Load()
	if evals == 0 {
		return ClassDead
	}
	if n.successCount.Load() == 0 {
		return ClassCold
	}
	if totalTicks > 0 {
		hitRate := float64(evals) / float64(totalTicks)
		if hitRate > 0.90 {
			return ClassHot
		}
		if hitRate < 0.01 {
			return ClassRare
		}
	}
	return ClassHealthy
}

// NodeReport holds metrics for a single node in the tree report.
type NodeReport struct {
	Name           string
	Classification Classification
	EvalCount      int64
	SuccessCount   int64
	FailCount      int64
	RunningCount   int64
}

// TreeReport aggregates metrics from all instrumented nodes in a tree.
type TreeReport struct {
	Nodes      []NodeReport
	TotalTicks int64
}

// DeadCount returns the number of dead (unreachable) nodes.
func (r *TreeReport) DeadCount() int {
	count := 0
	for _, n := range r.Nodes {
		if n.Classification == ClassDead {
			count++
		}
	}
	return count
}

// ColdCount returns the number of cold (never-succeeding) nodes.
func (r *TreeReport) ColdCount() int {
	count := 0
	for _, n := range r.Nodes {
		if n.Classification == ClassCold {
			count++
		}
	}
	return count
}

// InstrumentedTree holds a BT root with instrumentation on all nodes.
type InstrumentedTree struct {
	Root     bt.Node
	nodes    []*InstrumentedNode
	ticks    int64
	nameSeen map[string]int // dedup counter for repeated leaf names
}

// Tick advances the tree by one tick and increments the tick counter.
func (t *InstrumentedTree) Tick(ctx any) bt.Result {
	t.ticks++
	return t.Root.Tick(ctx)
}

// Report generates a TreeReport with classifications for all nodes.
func (t *InstrumentedTree) Report() *TreeReport {
	report := &TreeReport{
		Nodes:      make([]NodeReport, len(t.nodes)),
		TotalTicks: t.ticks,
	}
	for i, n := range t.nodes {
		report.Nodes[i] = NodeReport{
			Name:           n.Name,
			Classification: n.Classify(t.ticks),
			EvalCount:      n.EvalCount(),
			SuccessCount:   n.SuccessCount(),
			FailCount:      n.FailCount(),
			RunningCount:   n.RunningCount(),
		}
	}
	return report
}

// Node returns the InstrumentedNode with the given name, or nil.
func (t *InstrumentedTree) Node(name string) *InstrumentedNode {
	for _, n := range t.nodes {
		if n.Name == name {
			return n
		}
	}
	return nil
}

// InstrumentTree wraps all nodes in an existing BT with instrumentation.
// It recursively walks composites (which have exported Children fields)
// and wraps each node. Leaf nodes get positional names unless a name map
// is provided.
func InstrumentTree(root bt.Node) *InstrumentedTree {
	tree := &InstrumentedTree{nameSeen: make(map[string]int)}
	tree.Root = tree.instrumentNode(root, "root", 0)
	return tree
}

// dedupName appends a #N suffix if a name has been seen before.
func (t *InstrumentedTree) dedupName(name string) string {
	t.nameSeen[name]++
	if t.nameSeen[name] == 1 {
		return name
	}
	return name + "#" + itoa(t.nameSeen[name])
}

func (t *InstrumentedTree) instrumentNode(node bt.Node, prefix string, idx int) bt.Node {
	pos := prefix
	if idx > 0 {
		pos = prefix + "_" + itoa(idx)
	}

	switch n := node.(type) {
	case *bt.Selector:
		wrapped := make([]bt.Node, len(n.Children))
		for i, child := range n.Children {
			wrapped[i] = t.instrumentNode(child, pos+"/sel", i)
		}
		return bt.NewSelector(wrapped...)

	case *bt.Sequence:
		wrapped := make([]bt.Node, len(n.Children))
		for i, child := range n.Children {
			wrapped[i] = t.instrumentNode(child, pos+"/seq", i)
		}
		return bt.NewSequence(wrapped...)

	case *bt.ReactiveSelector:
		wrapped := make([]bt.Node, len(n.Children))
		for i, child := range n.Children {
			wrapped[i] = t.instrumentNode(child, pos+"/rsel", i)
		}
		return bt.NewReactiveSelector(wrapped...)

	case *bt.NamedNode:
		// Unwrap: instrument the inner node but use the human-readable label
		label := n.Label
		inner := n.Inner
		// If the inner is a composite (e.g. subtree), recurse into it
		switch inner.(type) {
		case *bt.Selector, *bt.Sequence, *bt.ReactiveSelector:
			return t.instrumentNode(inner, pos+"["+label+"]", 0)
		}
		// Leaf — use the label, disambiguated if repeated
		in := &InstrumentedNode{
			inner: inner,
			Name:  t.dedupName(label),
		}
		t.nodes = append(t.nodes, in)
		return in

	default:
		// Unnamed leaf node (Action, Condition, Inverter, etc.)
		in := &InstrumentedNode{
			inner: node,
			Name:  t.dedupName(pos),
		}
		t.nodes = append(t.nodes, in)
		return in
	}
}

// CloneTreeReport returns a deep copy of src with an independent Nodes slice,
// safe to use as a merge accumulator without mutating the source.
func CloneTreeReport(src *TreeReport) *TreeReport {
	nodes := make([]NodeReport, len(src.Nodes))
	copy(nodes, src.Nodes)
	return &TreeReport{Nodes: nodes, TotalTicks: src.TotalTicks}
}

// MergeTreeReport adds src's per-node counts and tick total into dst. Both must
// share the same tree topology (merged by node index); extra src nodes are
// ignored. Call ClassifyTreeReport afterwards to recompute classifications.
func MergeTreeReport(dst, src *TreeReport) {
	dst.TotalTicks += src.TotalTicks
	for i := range dst.Nodes {
		if i < len(src.Nodes) {
			dst.Nodes[i].EvalCount += src.Nodes[i].EvalCount
			dst.Nodes[i].SuccessCount += src.Nodes[i].SuccessCount
			dst.Nodes[i].FailCount += src.Nodes[i].FailCount
			dst.Nodes[i].RunningCount += src.Nodes[i].RunningCount
		}
	}
}

// ClassifyTreeReport recomputes each node's Classification from its merged counts.
func ClassifyTreeReport(r *TreeReport) {
	for i := range r.Nodes {
		n := &r.Nodes[i]
		n.Classification = ClassifyFromCounts(n.EvalCount, n.SuccessCount, r.TotalTicks)
	}
}

// ClassifyFromCounts determines classification given raw counts and total ticks.
func ClassifyFromCounts(evalCount, successCount, totalTicks int64) Classification {
	if evalCount == 0 {
		return ClassDead
	}
	if successCount == 0 {
		return ClassCold
	}
	if totalTicks > 0 {
		hitRate := float64(evalCount) / float64(totalTicks)
		if hitRate > 0.90 {
			return ClassHot
		}
		if hitRate < 0.01 {
			return ClassRare
		}
	}
	return ClassHealthy
}

func itoa(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return string(digits[i])
	}
	return itoa(i/10) + string(digits[i%10])
}
