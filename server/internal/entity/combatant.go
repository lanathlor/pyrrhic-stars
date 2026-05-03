package entity

// Caster is an entity that can cast abilities.
// Both Player and Enemy implement this interface.
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
// Both Player and Enemy implement this interface.
type Target interface {
	TargetID() uint16
	TargetPos() Vec3
	TargetAlive() bool
	TargetApplyDamage(amount float32) float32
}
