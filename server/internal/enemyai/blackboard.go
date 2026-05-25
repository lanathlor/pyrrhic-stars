package enemyai

import "slices"

// Blackboard provides per-entity BT memory: flags, counters, timers, and
// arbitrary typed values. Timers are decremented each tick via TickTimers.
type Blackboard struct {
	flags    map[string]bool
	counters map[string]int
	values   map[string]any

	// Timers stored as parallel slices to avoid map iteration allocation.
	timerKeys []string
	timerVals []float32
}

func NewBlackboard() *Blackboard {
	return &Blackboard{
		flags:    make(map[string]bool),
		counters: make(map[string]int),
		values:   make(map[string]any),
	}
}

// TickTimers decrements all active timers by dt. Call once at the start of
// each brain tick, before the tree runs. Uses slice iteration (zero allocs).
func (bb *Blackboard) TickTimers(dt float32) {
	n := 0
	for i, v := range bb.timerVals {
		v -= dt
		if v > 0 {
			bb.timerKeys[n] = bb.timerKeys[i]
			bb.timerVals[n] = v
			n++
		}
	}
	bb.timerKeys = bb.timerKeys[:n]
	bb.timerVals = bb.timerVals[:n]
}

// --- Flags ---

func (bb *Blackboard) GetFlag(key string) bool { return bb.flags[key] }
func (bb *Blackboard) SetFlag(key string)      { bb.flags[key] = true }
func (bb *Blackboard) ClearFlag(key string)    { delete(bb.flags, key) }

// --- Counters ---

func (bb *Blackboard) GetCounter(key string) int    { return bb.counters[key] }
func (bb *Blackboard) SetCounter(key string, v int) { bb.counters[key] = v }
func (bb *Blackboard) IncrementCounter(key string)  { bb.counters[key]++ }

// --- Timers ---

// StartTimer sets a timer that will expire after duration seconds.
func (bb *Blackboard) StartTimer(key string, duration float32) {
	// Update existing timer if present
	for i, k := range bb.timerKeys {
		if k == key {
			bb.timerVals[i] = duration
			return
		}
	}
	bb.timerKeys = append(bb.timerKeys, key)
	bb.timerVals = append(bb.timerVals, duration)
}

// TimerExpired returns true if the named timer does not exist (never started
// or already expired).
func (bb *Blackboard) TimerExpired(key string) bool {
	return !slices.Contains(bb.timerKeys, key)
}

// TimerRemaining returns the seconds left on a timer, or 0 if expired/absent.
func (bb *Blackboard) TimerRemaining(key string) float32 {
	for i, k := range bb.timerKeys {
		if k == key {
			return bb.timerVals[i]
		}
	}
	return 0
}

// --- Arbitrary values ---

func (bb *Blackboard) Set(key string, v any) { bb.values[key] = v }
func (bb *Blackboard) Get(key string) any    { return bb.values[key] }
func (bb *Blackboard) Delete(key string)     { delete(bb.values, key) }

func (bb *Blackboard) GetFloat32(key string) float32 {
	if v, ok := bb.values[key].(float32); ok {
		return v
	}
	return 0
}

func (bb *Blackboard) GetInt(key string) int {
	if v, ok := bb.values[key].(int); ok {
		return v
	}
	return 0
}

func (bb *Blackboard) GetString(key string) string {
	if v, ok := bb.values[key].(string); ok {
		return v
	}
	return ""
}

// Reset clears all blackboard state.
func (bb *Blackboard) Reset() {
	clear(bb.flags)
	clear(bb.counters)
	clear(bb.values)
	bb.timerKeys = bb.timerKeys[:0]
	bb.timerVals = bb.timerVals[:0]
}
