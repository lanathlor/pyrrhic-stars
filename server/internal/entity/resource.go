package entity

// Resource tracks a generic expendable resource (stamina, shield, etc.).
type Resource struct {
	Current    float32
	Max        float32
	Regen      float32 // per-second regen rate (negative = decay)
	RegenDelay float32 // seconds after spending before regen starts
	DelayTimer float32 // remaining delay countdown
}
