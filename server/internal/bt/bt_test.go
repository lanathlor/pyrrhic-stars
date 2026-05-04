package bt

import "testing"

// helper that returns a fixed result
func fixed(r Result) Node {
	return NewAction(func(any) Result { return r })
}

// counter tracks how many times a node was ticked
type counter struct {
	n    int
	node Node
}

func counted(node Node) *counter {
	c := &counter{node: node}
	return c
}

func (c *counter) Tick(ctx any) Result {
	c.n++
	return c.node.Tick(ctx)
}

func TestSelectorReturnsFirstSuccess(t *testing.T) {
	s := NewSelector(fixed(Failure), fixed(Success), fixed(Failure))
	if r := s.Tick(nil); r != Success {
		t.Fatalf("want Success, got %d", r)
	}
}

func TestSelectorReturnsFailureWhenAllFail(t *testing.T) {
	s := NewSelector(fixed(Failure), fixed(Failure))
	if r := s.Tick(nil); r != Failure {
		t.Fatalf("want Failure, got %d", r)
	}
}

func TestSelectorResumesFromRunningChild(t *testing.T) {
	calls := 0
	runOnce := NewAction(func(any) Result {
		calls++
		if calls == 1 {
			return Running
		}
		return Success
	})
	skipped := counted(fixed(Failure))
	s := NewSelector(skipped, runOnce)

	// First tick: child 0 fails, child 1 returns Running
	if r := s.Tick(nil); r != Running {
		t.Fatalf("tick 1: want Running, got %d", r)
	}
	// Second tick: should resume from child 1, skip child 0
	skipped.n = 0
	if r := s.Tick(nil); r != Success {
		t.Fatalf("tick 2: want Success, got %d", r)
	}
	if skipped.n != 0 {
		t.Fatalf("child 0 should not be re-ticked after resume, got %d ticks", skipped.n)
	}
}

func TestSelectorResetsAfterSuccess(t *testing.T) {
	calls := 0
	child := NewAction(func(any) Result {
		calls++
		if calls == 1 {
			return Running
		}
		return Success
	})
	first := counted(fixed(Failure))
	s := NewSelector(first, child)

	s.Tick(nil) // child 0 fail, child 1 running
	s.Tick(nil) // resume child 1 -> success, reset

	// Third tick should start from child 0 again
	first.n = 0
	s.Tick(nil)
	if first.n != 1 {
		t.Fatalf("after reset, child 0 should be ticked, got %d", first.n)
	}
}

func TestSequenceReturnsSuccessWhenAllSucceed(t *testing.T) {
	s := NewSequence(fixed(Success), fixed(Success), fixed(Success))
	if r := s.Tick(nil); r != Success {
		t.Fatalf("want Success, got %d", r)
	}
}

func TestSequenceReturnsFirstFailure(t *testing.T) {
	third := counted(fixed(Success))
	s := NewSequence(fixed(Success), fixed(Failure), third)
	if r := s.Tick(nil); r != Failure {
		t.Fatalf("want Failure, got %d", r)
	}
	if third.n != 0 {
		t.Fatal("child after failure should not be ticked")
	}
}

func TestSequenceResumesFromRunningChild(t *testing.T) {
	calls := 0
	runOnce := NewAction(func(any) Result {
		calls++
		if calls == 1 {
			return Running
		}
		return Success
	})
	first := counted(fixed(Success))
	last := counted(fixed(Success))
	s := NewSequence(first, runOnce, last)

	// Tick 1: child 0 success, child 1 running
	if r := s.Tick(nil); r != Running {
		t.Fatalf("tick 1: want Running, got %d", r)
	}
	if last.n != 0 {
		t.Fatal("child 2 should not be ticked yet")
	}

	// Tick 2: resume from child 1 (success), then child 2 (success)
	first.n = 0
	if r := s.Tick(nil); r != Success {
		t.Fatalf("tick 2: want Success, got %d", r)
	}
	if first.n != 0 {
		t.Fatal("child 0 should not be re-ticked after resume")
	}
	if last.n != 1 {
		t.Fatalf("child 2 should be ticked once, got %d", last.n)
	}
}

func TestInverterFlipsSuccessAndFailure(t *testing.T) {
	tests := []struct {
		input Result
		want  Result
	}{
		{Success, Failure},
		{Failure, Success},
		{Running, Running},
	}
	for _, tt := range tests {
		inv := NewInverter(fixed(tt.input))
		if got := inv.Tick(nil); got != tt.want {
			t.Errorf("Inverter(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestNestedSelectorInSequence(t *testing.T) {
	// Sequence: [Success, Selector[Failure, Success]]
	inner := NewSelector(fixed(Failure), fixed(Success))
	s := NewSequence(fixed(Success), inner)
	if r := s.Tick(nil); r != Success {
		t.Fatalf("want Success, got %d", r)
	}
}

func TestNestedSequenceInSelector(t *testing.T) {
	// Selector: [Sequence[Success, Failure], Success]
	inner := NewSequence(fixed(Success), fixed(Failure))
	s := NewSelector(inner, fixed(Success))
	if r := s.Tick(nil); r != Success {
		t.Fatalf("want Success, got %d", r)
	}
}

func TestTreeReset(t *testing.T) {
	calls := 0
	child := NewAction(func(any) Result {
		calls++
		return Running
	})
	tree := NewTree(NewSequence(fixed(Success), child))

	tree.Tick(nil) // child 0 success, child 1 running
	tree.Reset()

	// After reset, should start from child 0 again
	first := counted(fixed(Success))
	tree.Root.(*Sequence).Children[0] = first
	tree.Tick(nil)
	if first.n != 1 {
		t.Fatal("after reset, child 0 should be re-ticked")
	}
}

func TestConditionNode(t *testing.T) {
	trueNode := NewCondition(func(any) bool { return true })
	falseNode := NewCondition(func(any) bool { return false })

	if r := trueNode.Tick(nil); r != Success {
		t.Fatalf("true condition: want Success, got %d", r)
	}
	if r := falseNode.Tick(nil); r != Failure {
		t.Fatalf("false condition: want Failure, got %d", r)
	}
}

func TestEmptySelector(t *testing.T) {
	s := NewSelector()
	if r := s.Tick(nil); r != Failure {
		t.Fatalf("empty selector: want Failure, got %d", r)
	}
}

func TestEmptySequence(t *testing.T) {
	s := NewSequence()
	if r := s.Tick(nil); r != Success {
		t.Fatalf("empty sequence: want Success, got %d", r)
	}
}

func TestSelectorRunningChildFailsOnResume(t *testing.T) {
	calls := 0
	child := NewAction(func(any) Result {
		calls++
		if calls == 1 {
			return Running
		}
		return Failure
	})
	fallback := fixed(Success)
	s := NewSelector(child, fallback)

	s.Tick(nil) // child 0 running
	// Tick 2: child 0 fails, should continue to child 1
	if r := s.Tick(nil); r != Success {
		t.Fatalf("want Success from fallback, got %d", r)
	}
}

func TestSequenceRunningChildFailsOnResume(t *testing.T) {
	calls := 0
	child := NewAction(func(any) Result {
		calls++
		if calls == 1 {
			return Running
		}
		return Failure
	})
	s := NewSequence(fixed(Success), child, fixed(Success))

	s.Tick(nil) // child 0 success, child 1 running
	if r := s.Tick(nil); r != Failure {
		t.Fatalf("want Failure, got %d", r)
	}
}

// --- ReactiveSelector tests ---

func TestReactiveSelectorAlwaysEvaluatesFromFirst(t *testing.T) {
	highPriority := counted(fixed(Failure))
	lowPriority := fixed(Running)
	s := NewReactiveSelector(highPriority, lowPriority)

	s.Tick(nil) // high fails, low running
	s.Tick(nil) // should re-evaluate high first
	s.Tick(nil)
	if highPriority.n != 3 {
		t.Fatalf("high priority should be ticked every time, got %d", highPriority.n)
	}
}

func TestReactiveSelectorHighPriorityInterruptsLow(t *testing.T) {
	ticks := 0
	// High priority: fails first 2 ticks, then succeeds
	high := NewAction(func(any) Result {
		ticks++
		if ticks <= 2 {
			return Failure
		}
		return Success
	})
	lowTicks := 0
	low := NewAction(func(any) Result {
		lowTicks++
		return Running
	})
	s := NewReactiveSelector(high, low)

	// Tick 1: high fails, low Running
	if r := s.Tick(nil); r != Running {
		t.Fatalf("tick 1: want Running, got %d", r)
	}
	// Tick 2: high fails, low Running (resumed because reactive re-evaluates from 0)
	if r := s.Tick(nil); r != Running {
		t.Fatalf("tick 2: want Running, got %d", r)
	}
	if lowTicks != 2 {
		t.Fatalf("low should be ticked twice, got %d", lowTicks)
	}
	// Tick 3: high succeeds — interrupts low
	if r := s.Tick(nil); r != Success {
		t.Fatalf("tick 3: want Success, got %d", r)
	}
}

func TestReactiveSelectorResetsInterruptedChild(t *testing.T) {
	// Sequence that tracks running state
	inner := NewSequence(fixed(Success), fixed(Running))
	// High priority starts failing, then succeeds
	calls := 0
	high := NewAction(func(any) Result {
		calls++
		if calls <= 1 {
			return Failure
		}
		return Success
	})
	s := NewReactiveSelector(high, inner)

	// Tick 1: high fails, inner Sequence runs (child 0 success, child 1 running)
	s.Tick(nil)
	if inner.runningIdx != 1 {
		t.Fatalf("inner should have runningIdx=1, got %d", inner.runningIdx)
	}

	// Tick 2: high succeeds — inner should be reset
	s.Tick(nil)
	if inner.runningIdx != -1 {
		t.Fatalf("inner should be reset after interruption, runningIdx=%d", inner.runningIdx)
	}
}

func TestReactiveSelectorEmptyReturnsFailure(t *testing.T) {
	s := NewReactiveSelector()
	if r := s.Tick(nil); r != Failure {
		t.Fatalf("empty reactive selector: want Failure, got %d", r)
	}
}
