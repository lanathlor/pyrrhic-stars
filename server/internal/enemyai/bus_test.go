package enemyai

import (
	"testing"

	"codex-online/server/internal/entity"
)

func busEnemy(id uint16, groupID int) *entity.Enemy {
	e := entity.NewEnemy(id, 100, "test_mob")
	e.GroupID = groupID
	return e
}

// TestBus_CountRecent_WindowAndSelfExclusion verifies the sliding window and
// that a listener never hears its own events.
func TestBus_CountRecent_WindowAndSelfExclusion(t *testing.T) {
	b := NewBus()
	e1, e2 := busEnemy(1, 7), busEnemy(2, 7)
	enemies := []*entity.Enemy{e1, e2}

	b.BeginTick(10, 0.05)
	b.RebuildClusters(enemies)
	b.Emit(ChanCommitStarted, 1, "test_mob", "bolt")

	// Same tick: e2 hears e1, e1 does not hear itself.
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 2); got != 1 {
		t.Errorf("e2 CountRecent = %d, want 1", got)
	}
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 1); got != 0 {
		t.Errorf("e1 CountRecent (self) = %d, want 0", got)
	}

	// 0.5s window = 10 ticks: event at tick 10 is audible through tick 19,
	// gone at tick 20.
	b.BeginTick(19, 0.05)
	b.RebuildClusters(enemies)
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 2); got != 1 {
		t.Errorf("tick 19: CountRecent = %d, want 1", got)
	}
	b.BeginTick(20, 0.05)
	b.RebuildClusters(enemies)
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 2); got != 0 {
		t.Errorf("tick 20: CountRecent = %d, want 0 (window expired)", got)
	}
}

// TestBus_CountRecent_DefFilter verifies def-scoped counting ("stagger against
// mobs like me"): a ranged mob's gate must not be held by melee commits.
func TestBus_CountRecent_DefFilter(t *testing.T) {
	b := NewBus()
	ranged := entity.NewEnemy(1, 100, "ranged_mob")
	ranged.GroupID = 7
	melee := entity.NewEnemy(2, 100, "melee_mob")
	melee.GroupID = 7
	enemies := []*entity.Enemy{ranged, melee}

	b.BeginTick(1, 0.05)
	b.RebuildClusters(enemies)
	b.Emit(ChanCommitStarted, 2, "melee_mob", "slash")

	if got := b.CountRecent(ChanCommitStarted, "ranged_mob", 0.5, 1); got != 0 {
		t.Errorf("def-filtered CountRecent = %d, want 0 (melee commit must not count)", got)
	}
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 1); got != 1 {
		t.Errorf("unfiltered CountRecent = %d, want 1", got)
	}
}

// TestBus_Clusters_GroupAndThreatLinks verifies cluster derivation: same
// GroupID links mobs, a shared aggroed player links across groups, and an
// unrelated fight stays inaudible.
func TestBus_Clusters_GroupAndThreatLinks(t *testing.T) {
	b := NewBus()
	a1, a2 := busEnemy(1, 10), busEnemy(2, 10) // pack A
	b1 := busEnemy(3, 20)                      // pack B
	far := busEnemy(4, 30)                     // unrelated fight
	enemies := []*entity.Enemy{a1, a2, b1, far}

	b.BeginTick(1, 0.05)
	b.RebuildClusters(enemies)

	if !b.SameCluster(1, 2) {
		t.Error("same GroupID should share a cluster")
	}
	if b.SameCluster(1, 3) {
		t.Error("distinct packs with no shared aggro should not share a cluster")
	}

	// Player 9 chain-pulls: has threat on pack A and pack B → clusters merge.
	a1.AddThreat(9, 10)
	b1.AddThreat(9, 5)
	b.RebuildClusters(enemies)
	if !b.SameCluster(1, 3) {
		t.Error("shared aggroed player should merge pack clusters")
	}
	if !b.SameCluster(2, 3) {
		t.Error("merge should be transitive through GroupID")
	}
	if b.SameCluster(4, 1) {
		t.Error("unrelated fight must stay in its own cluster")
	}

	// Events across the merged cluster are audible; the unrelated mob hears nothing.
	b.Emit(ChanCommitStarted, 1, "test_mob", "bolt")
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 3); got != 1 {
		t.Errorf("merged cluster: CountRecent = %d, want 1", got)
	}
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 4); got != 0 {
		t.Errorf("unrelated mob: CountRecent = %d, want 0", got)
	}
}

// TestBus_DeadSourceStaysAudible verifies a dead mob's recent events remain
// audible (needed for future ally_died reactions and lease auto-release).
func TestBus_DeadSourceStaysAudible(t *testing.T) {
	b := NewBus()
	e1, e2 := busEnemy(1, 7), busEnemy(2, 7)
	enemies := []*entity.Enemy{e1, e2}

	b.BeginTick(1, 0.05)
	b.RebuildClusters(enemies)
	b.Emit(ChanCommitStarted, 1, "test_mob", "bolt")

	e1.Alive = false
	b.BeginTick(2, 0.05)
	b.RebuildClusters(enemies)
	if got := b.CountRecent(ChanCommitStarted, "", 0.5, 2); got != 1 {
		t.Errorf("dead source: CountRecent = %d, want 1", got)
	}
}

// TestBus_Retention verifies production pruning keeps the log bounded while
// RetainAll keeps everything for simulations.
func TestBus_Retention(t *testing.T) {
	b := NewBus()
	b.BeginTick(1, 0.05)
	b.Emit(ChanCommitStarted, 1, "test_mob", "bolt")

	// Within retention: still there.
	b.BeginTick(1+busRetainTicks, 0.05)
	if len(b.Events()) != 1 {
		t.Fatalf("events = %d, want 1 (inside retention)", len(b.Events()))
	}
	// Past retention: pruned.
	b.BeginTick(2+busRetainTicks, 0.05)
	if len(b.Events()) != 0 {
		t.Fatalf("events = %d, want 0 (pruned)", len(b.Events()))
	}

	sim := NewBus()
	sim.RetainAll = true
	sim.BeginTick(1, 0.05)
	sim.Emit(ChanCommitStarted, 1, "test_mob", "bolt")
	sim.BeginTick(5000, 0.05)
	if len(sim.Events()) != 1 {
		t.Fatalf("RetainAll events = %d, want 1", len(sim.Events()))
	}
}
