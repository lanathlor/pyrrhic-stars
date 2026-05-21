package entity

// HealingZone is a persistent ground-effect area that periodically heals
// players standing inside it.
type HealingZone struct {
	ID          uint32
	OwnerID     uint16  // peer ID of the player who placed this zone
	Position    Vec3    // center of the zone
	Radius      float32 // horizontal radius (XZ plane)
	HealPerTick float32 // HP restored each heal pulse
	Duration    float32 // remaining lifetime in seconds
	TickTimer   float32 // countdown to next heal pulse
	Interval    float32 // seconds between heal pulses
	AbilityID   string  // originating ability (for combat log / UI)
}

// Tick advances the zone's lifetime by dt seconds.
// Returns true when the zone has expired and should be removed.
func (z *HealingZone) Tick(dt float32) bool {
	z.Duration -= dt
	return z.Duration <= 0
}

// ShouldTick advances the heal pulse timer and returns true when a heal
// pulse should fire this frame.
func (z *HealingZone) ShouldTick(dt float32) bool {
	z.TickTimer -= dt
	if z.TickTimer <= 0 {
		z.TickTimer += z.Interval
		return true
	}
	return false
}

// ContainsPoint returns true if pos falls within the zone's horizontal
// radius (Y is ignored — the check is on the XZ plane only).
func (z *HealingZone) ContainsPoint(pos Vec3) bool {
	dx := pos.X - z.Position.X
	dz := pos.Z - z.Position.Z
	return dx*dx+dz*dz <= z.Radius*z.Radius
}
