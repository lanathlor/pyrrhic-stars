package ability

import (
	"fmt"
	"testing"
)

func TestVitalDrainSustainLifecycle(t *testing.T) {
	eng := NewEngine(nil)
	def := eng.GetAbility("vital_drain")
	if def == nil {
		t.Fatal("vital_drain not registered")
	}
	t.Logf("def: CommitTime=%.1f Sustain=%v Hit.Type=%d HitNearestN=%d match=%v",
		def.CommitTime, def.Sustain, def.Hit.Type, HitNearestN, def.Hit.Type == HitNearestN)

	runner := &PlayerAbilityRunner{}
	runner.Start(def)
	t.Logf("after Start: phase=%d (Commit=%d)", runner.Phase, PRunnerCommit)

	// Tick through 1.0s commit + 0.1s execute = 1.1s
	for i := 0; i < 24; i++ {
		runner.Tick(0.05)
	}
	t.Logf("after 1.2s: phase=%d (Sustain=%d)", runner.Phase, PRunnerSustain)
	if runner.Phase != PRunnerSustain {
		t.Fatalf("expected Sustain phase, got %d", runner.Phase)
	}

	// Tick sustain and count fires
	fires := 0
	for i := 0; i < 20; i++ {
		if runner.Tick(0.05) {
			fires++
			t.Logf("  sustain tick %d at elapsed=%.2f", fires, runner.SustainElapsed)
		}
	}
	t.Logf("sustain fires in 1.0s: %d", fires)
	if fires == 0 {
		t.Fatal("expected sustain ticks to fire")
	}
	fmt.Println("OK: vital_drain sustain ticks fire correctly")
}
