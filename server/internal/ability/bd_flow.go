package ability

import "codex-online/server/internal/entity"

// Flow tuning constants.
const (
	flowBaseWindow     float32 = 4.0  // seconds
	flowExtendPerChain float32 = 2.0  // seconds added per unique transition
	flowMaxWindow      float32 = 12.0 // hard cap on window duration
	flowBonusPerChain  float32 = 0.05 // 5% per chain step
	flowEmpoweredThresh        = 3
	flowMaximumThresh           = 6
)

// FlowState tracks the Blade Dancer's Flow mastery chain.
// Each unique configuration transition within a window extends the chain
// and amplifies the next transition's damage. Repeating a transition breaks it.
type FlowState struct {
	UsedTransitions uint32  // bitset: bit (origin*5+dest) marks used transitions
	ChainLen        int     // number of unique transitions in current chain
	Timer           float32 // remaining window before chain expires
}

// transitionBit returns the bitmask for an origin->dest transition.
func transitionBit(origin, dest int) uint32 {
	return 1 << uint(origin*5+dest)
}

// HasTransition returns true if origin->dest has been used in the current chain.
func (s *FlowState) HasTransition(origin, dest int) bool {
	return s.UsedTransitions&transitionBit(origin, dest) != 0
}

// RecordTransition attempts to add a transition to the chain.
// Returns the damage multiplier for this cast.
// If the transition is a repeat, resets the chain and returns 1.0 (no bonus).
func (s *FlowState) RecordTransition(origin, dest int, mastery float32) float32 {
	bit := transitionBit(origin, dest)
	if s.UsedTransitions&bit != 0 {
		// Repeat transition -- break the chain
		s.Reset()
		return 1.0
	}

	// Unique transition -- compute bonus from current chain length (before increment)
	mult := 1.0 + float32(s.ChainLen)*flowBonusPerChain*(1.0+mastery/100.0)

	// Extend chain
	s.UsedTransitions |= bit
	s.ChainLen++

	// Refresh timer
	window := flowBaseWindow + flowExtendPerChain*float32(s.ChainLen)
	if window > flowMaxWindow {
		window = flowMaxWindow
	}
	s.Timer = window

	return mult
}

// Tick decrements the timer by dt. Resets if timer expires.
func (s *FlowState) Tick(dt float32) {
	if s.Timer <= 0 {
		return
	}
	s.Timer -= dt
	if s.Timer <= 0 {
		s.Reset()
	}
}

// Reset zeroes all state.
func (s *FlowState) Reset() {
	s.UsedTransitions = 0
	s.ChainLen = 0
	s.Timer = 0
}

// Tier returns the current Flow tier (reuses Onslaught tier constants).
func (s *FlowState) Tier() uint8 {
	if s.ChainLen >= flowMaximumThresh {
		return TierMaximum
	}
	if s.ChainLen >= flowEmpoweredThresh {
		return TierEmpowered
	}
	return TierStandard
}

// StackCount returns the chain length (used by codec via interface).
func (s *FlowState) StackCount() int {
	return s.ChainLen
}

// DamageMult returns the current damage multiplier without recording a transition.
func (s *FlowState) DamageMult(mastery float32) float32 {
	return 1.0 + float32(s.ChainLen)*flowBonusPerChain*(1.0+mastery/100.0)
}

// getFlowState returns the flow state for a BD player, creating it if needed.
func getFlowState(p *entity.Player) *FlowState {
	if s, ok := p.AbilityState["flow"].(*FlowState); ok {
		return s
	}
	s := &FlowState{}
	p.AbilityState["flow"] = s
	return s
}

// bdFlowTick is the tick handler that expires the Flow timer.
func bdFlowTick(_ *Engine, p *entity.Player, dt float32, _ *TickContext) []DamageResult {
	if p.ClassID != entity.ClassBladeDancer {
		return nil
	}
	if s, ok := p.AbilityState["flow"].(*FlowState); ok {
		s.Tick(dt)
	}
	return nil
}
