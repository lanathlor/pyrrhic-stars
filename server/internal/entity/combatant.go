package entity

import "math"

// Combatant holds the fields shared by all combat-participating entities
// (Player, Enemy, and future types like turrets or summons).
// Embed this struct to get spatial, health, and identity fields for free.
type Combatant struct {
	ID        uint16
	Position  Vec3
	RotationY float32
	Velocity  Vec3
	Health    float32
	MaxHealth float32
	Alive     bool
}

// Forward returns the unit vector in the direction the entity is facing
// (Godot convention: -Z is forward).
func (c *Combatant) Forward() Vec3 {
	s := float32(math.Sin(float64(c.RotationY)))
	co := float32(math.Cos(float64(c.RotationY)))
	return Vec3{-s, 0, -co}
}

// EyePos returns the position offset upward by the given eye height.
func (c *Combatant) EyePos(height float32) Vec3 {
	return c.Position.Add(Vec3{Y: height})
}

// --- Shared Caster/Target method implementations ---
// These are promoted into Player/Enemy via embedding.
// Override in the outer type where behavior differs.

func (c *Combatant) CasterID() uint16    { return c.ID }
func (c *Combatant) CasterPos() Vec3     { return c.Position }
func (c *Combatant) CasterForward() Vec3 { return c.Forward() }
func (c *Combatant) CasterAlive() bool   { return c.Alive }
func (c *Combatant) TargetID() uint16    { return c.ID }
func (c *Combatant) TargetPos() Vec3     { return c.Position }
func (c *Combatant) TargetAlive() bool   { return c.Alive }

// Caster is an entity that can cast abilities.
type Caster interface {
	CasterID() uint16
	CasterPos() Vec3
	CasterForward() Vec3
	CasterEyePos() Vec3
	CasterAimDir() Vec3
	CasterAlive() bool
	CasterDamageMult() float32
}

// Target is an entity that can receive damage from abilities.
type Target interface {
	TargetID() uint16
	TargetPos() Vec3
	TargetAlive() bool
	TargetApplyDamage(amount float32) float32
}

// Threateable is a Target that tracks threat from damage sources.
// Use this instead of type-asserting to *Enemy when you only need AddThreat.
type Threateable interface {
	Target
	AddThreat(peerID uint16, amount float32)
}
