package ability

import "codex-online/server/internal/entity"

// Devotion tier thresholds (Shield Vanguard mastery).
const (
	devotionEmpoweredThreshold float32 = 30
	devotionMaximumThreshold   float32 = 60
)

// DevotionState tracks accumulated charges from blocking damage.
// Charges are consumed by Retaliate for burst damage.
type DevotionState struct {
	Charges float32
}

// StackCount returns the integer charge count (codec compatibility).
func (s *DevotionState) StackCount() int {
	return int(s.Charges)
}

// Tier returns the current empowerment tier based on charge thresholds.
func (s *DevotionState) Tier() uint8 {
	if s.Charges >= devotionMaximumThreshold {
		return TierMaximum
	}
	if s.Charges >= devotionEmpoweredThreshold {
		return TierEmpowered
	}
	return TierStandard
}

// AddCharges converts absorbed damage into Devotion charges.
// Mastery stat governs the conversion rate.
func (s *DevotionState) AddCharges(absorbed, mastery float32) {
	s.Charges += absorbed * (0.15 + mastery/500.0)
}

// ConsumeAll returns all accumulated charges and zeroes the pool.
func (s *DevotionState) ConsumeAll() float32 {
	c := s.Charges
	s.Charges = 0
	return c
}

// Reset zeroes all charges.
func (s *DevotionState) Reset() {
	s.Charges = 0
}

// getDevotionState returns the devotion state for a player, creating it if needed.
func getDevotionState(p *entity.Player) *DevotionState {
	if s, ok := p.AbilityState["devotion"].(*DevotionState); ok {
		return s
	}
	s := &DevotionState{}
	p.AbilityState["devotion"] = s
	return s
}
