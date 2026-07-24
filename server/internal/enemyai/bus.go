package enemyai

import (
	"slices"

	"codex-online/server/internal/entity"
)

// Channel names for system-emitted coordination events. Tree-authored
// announce(...) leaves may use any custom channel name.
const (
	// ChanCommitStarted is emitted by the AbilityRunner whenever an enemy
	// commits to an ability (telegraph start).
	ChanCommitStarted = "commit_started"
)

// busRetainTicks bounds the production log to the recent past. No consumer may
// read further back than its query window, so anything older is semantically
// dead. 600 ticks = 30s at 20Hz, far beyond any heard/clear_to_fire window.
const busRetainTicks = 600

// BusEvent is one entry in the zone coordination log: a fact that something
// happened, tagged with who and when. Events are signals, not messages: they
// carry no payload beyond an optional ability ID.
type BusEvent struct {
	Tick      uint32
	Source    uint16 // enemy entity ID
	SourceDef string // enemy def name (debug dumps, test filters, def-scoped queries)
	Channel   string
	Ability   string // ability ID for commit_started, "" otherwise
}

// Bus is the zone-wide coordination event log. One per zone; all enemy brains
// in the zone append to and query the same log. Design rules:
//
//   - Facts only, no state: coordination state (who fired recently) is always
//     a query over a recent window, never a stored flag.
//   - Audibility is scoped by combat cluster: a listener only hears events
//     whose source shares its cluster (same GroupID, or transitively sharing
//     an aggroed player via threat tables). Clusters are derived per tick,
//     never stored, so chain-pulled packs merge and split for free.
//   - Coordination may only delay an action within a bounded window, never
//     require another mob's action (no deadlock by construction).
//   - No consumer may depend on old events: retention is a short ring, and
//     the leaf vocabulary must never grow a "did X ever happen" condition.
//
// Everything is synchronous inside the zone tick: no goroutines, no locks.
type Bus struct {
	// RetainAll keeps the full log instead of pruning (simulations keep it
	// for coordination assertions and debug timelines).
	RetainAll bool

	events []BusEvent
	tick   uint32
	dt     float32

	// Cluster derivation state, rebuilt each tick by RebuildClusters.
	clusterOf map[uint16]int // enemy ID → cluster root (index into parent)
	parent    []int          // union-find buffer
}

// NewBus creates an empty coordination bus.
func NewBus() *Bus {
	return &Bus{
		dt:        0.05,
		clusterOf: make(map[uint16]int),
	}
}

// BeginTick advances the bus clock and prunes events older than the retention
// window. Call once per zone tick, before any brain runs.
func (b *Bus) BeginTick(tick uint32, dt float32) {
	b.tick = tick
	if dt > 0 {
		b.dt = dt
	}
	if b.RetainAll {
		return
	}
	cut := 0
	for cut < len(b.events) && b.events[cut].Tick+busRetainTicks < tick {
		cut++
	}
	if cut > 0 {
		b.events = append(b.events[:0], b.events[cut:]...)
	}
}

// Emit appends an event at the current tick.
func (b *Bus) Emit(channel string, source uint16, sourceDef, abilityID string) {
	b.events = append(b.events, BusEvent{
		Tick:      b.tick,
		Source:    source,
		SourceDef: sourceDef,
		Channel:   channel,
		Ability:   abilityID,
	})
}

// Events returns the raw log (retained window, or everything under RetainAll).
func (b *Bus) Events() []BusEvent { return b.events }

// CountRecent counts events on channel within the last `window` seconds that
// the listener can hear: emitted by another mob in the listener's combat
// cluster. sourceDef, when non-empty, restricts to events from that mob def
// ("stagger against mobs like me").
func (b *Bus) CountRecent(channel, sourceDef string, window float32, listener uint16) int {
	windowTicks := uint32(window / b.dt)
	n := 0
	for _, ev := range slices.Backward(b.events) {
		if b.tick-ev.Tick >= windowTicks {
			break // events are tick-ordered; everything earlier is out of window
		}
		if ev.Channel != channel || ev.Source == listener {
			continue
		}
		if sourceDef != "" && ev.SourceDef != sourceDef {
			continue
		}
		if !b.SameCluster(ev.Source, listener) {
			continue
		}
		n++
	}
	return n
}

// RebuildClusters derives combat clusters from current aggro state: enemies
// sharing a GroupID or (transitively) an aggroed player land in one cluster.
// Dead enemies keep their membership so their recent events stay audible.
// Call once per zone tick, after BeginTick.
func (b *Bus) RebuildClusters(enemies []*entity.Enemy) {
	clear(b.clusterOf)
	b.parent = b.parent[:0]

	// Assign each enemy an index and self-parent.
	idx := make(map[uint16]int, len(enemies))
	live := b.parent
	for _, e := range enemies {
		if e == nil {
			continue
		}
		idx[e.ID] = len(live)
		live = append(live, len(live))
	}
	b.parent = live

	// Union by shared GroupID and by shared threat-table player.
	groupRoot := make(map[int]int)
	playerRoot := make(map[uint16]int)
	i := 0
	for _, e := range enemies {
		if e == nil {
			continue
		}
		if e.GroupID != 0 {
			if r, ok := groupRoot[e.GroupID]; ok {
				b.union(r, i)
			} else {
				groupRoot[e.GroupID] = i
			}
		}
		for pid := range e.ThreatTable {
			if r, ok := playerRoot[pid]; ok {
				b.union(r, i)
			} else {
				playerRoot[pid] = i
			}
		}
		i++
	}

	for _, e := range enemies {
		if e == nil {
			continue
		}
		b.clusterOf[e.ID] = b.find(idx[e.ID])
	}
}

// SameCluster reports whether two enemies share a combat cluster. Unknown IDs
// (never seen by RebuildClusters) hear nothing and are heard by no one.
func (b *Bus) SameCluster(a, c uint16) bool {
	ra, ok := b.clusterOf[a]
	if !ok {
		return false
	}
	rc, ok := b.clusterOf[c]
	return ok && ra == rc
}

// --- union-find ---

func (b *Bus) find(i int) int {
	for b.parent[i] != i {
		b.parent[i] = b.parent[b.parent[i]] // path halving
		i = b.parent[i]
	}
	return i
}

func (b *Bus) union(i, j int) {
	ri, rj := b.find(i), b.find(j)
	if ri != rj {
		b.parent[rj] = ri
	}
}
