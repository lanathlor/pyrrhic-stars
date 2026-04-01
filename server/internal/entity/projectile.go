package entity

// Projectile is a server-side projectile entity.
type Projectile struct {
	ID        uint32
	OwnerID   uint16 // 0 = enemy-owned
	Position  Vec3
	Direction Vec3
	Speed     float32
	Damage    float32
	Lifetime  float32
	Timer     float32
	Alive     bool
}

// NewProjectile creates a projectile.
func NewProjectile(id uint32, ownerID uint16, pos, dir Vec3, speed, damage, lifetime float32) *Projectile {
	return &Projectile{
		ID:        id,
		OwnerID:   ownerID,
		Position:  pos,
		Direction: dir.Normalized(),
		Speed:     speed,
		Damage:    damage,
		Lifetime:  lifetime,
		Alive:     true,
	}
}

// Tick advances the projectile by dt seconds.
func (p *Projectile) Tick(dt float32) {
	if !p.Alive {
		return
	}
	p.Position = p.Position.Add(p.Direction.Scale(p.Speed * dt))
	p.Timer += dt
	if p.Timer >= p.Lifetime {
		p.Alive = false
	}
}

// HitRadius is the collision radius for projectile hit checks.
const ProjectileHitRadius float32 = 0.5
