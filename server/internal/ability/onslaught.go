package ability

import "codex-online/server/internal/entity"

// Onslaught tier thresholds.
const (
	onslaughtEmpoweredThreshold = 3
	onslaughtMaximumThreshold   = 6
)

// Onslaught tier constants (encoded in buff_flags bits 6-7).
const (
	TierStandard  uint8 = 0
	TierEmpowered uint8 = 1
	TierMaximum   uint8 = 2
)

// OnslaughtState tracks consecutive landed attacks without taking damage.
type OnslaughtState struct {
	Stacks int
}

// StackCount returns the current stack count (used by codec via interface).
func (s *OnslaughtState) StackCount() int {
	return s.Stacks
}

// Tier returns the current empowerment tier.
func (s *OnslaughtState) Tier() uint8 {
	if s.Stacks >= onslaughtMaximumThreshold {
		return TierMaximum
	}
	if s.Stacks >= onslaughtEmpoweredThreshold {
		return TierEmpowered
	}
	return TierStandard
}

// Reset zeroes the stack counter (called when player takes damage).
func (s *OnslaughtState) Reset() {
	s.Stacks = 0
}

// Increment adds n stacks (one per enemy hit).
func (s *OnslaughtState) Increment(n int) {
	s.Stacks += n
}

// DamageMult returns the bonus damage multiplier from Onslaught stacks.
// ~3% per stack, scaled by Mastery stat.
func (s *OnslaughtState) DamageMult(mastery float32) float32 {
	return 1.0 + float32(s.Stacks)*0.03*(1.0+mastery/100.0)
}

// getOnslaughtState returns the onslaught state for a player, creating it if needed.
func getOnslaughtState(p *entity.Player) *OnslaughtState {
	if s, ok := p.AbilityState["onslaught"].(*OnslaughtState); ok {
		return s
	}
	s := &OnslaughtState{}
	p.AbilityState["onslaught"] = s
	return s
}
