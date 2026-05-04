package entity

import "math"

// Projectile is a server-side projectile entity.
type Projectile struct {
	ID        uint32
	OwnerID   uint16 // 0 = enemy-owned, >0 = player peer ID
	EnemyIdx  int    // index into World.Enemies (-1 = player-owned)
	Position  Vec3
	Direction Vec3
	Speed     float32
	Damage    float32
	Lifetime  float32
	Timer     float32
	Alive     bool

	// Pattern engine fields
	VisualTag       string  // ability name for client VFX selection
	Acceleration    float32 // speed change per second (neg = decelerate)
	AngularVelocity float32 // radians/s rotation of flight direction
	MaxSpeed        float32 // speed cap (0 = no cap)
}

// NewProjectile creates a projectile with linear motion.
func NewProjectile(id uint32, ownerID uint16, enemyIdx int, pos, dir Vec3, speed, damage, lifetime float32) *Projectile {
	return &Projectile{
		ID:        id,
		OwnerID:   ownerID,
		EnemyIdx:  enemyIdx,
		Position:  pos,
		Direction: dir.Normalized(),
		Speed:     speed,
		Damage:    damage,
		Lifetime:  lifetime,
		Alive:     true,
	}
}

// Tick advances the projectile by dt seconds.
// Applies acceleration and angular velocity for curved/accelerating projectiles.
func (p *Projectile) Tick(dt float32) {
	if !p.Alive {
		return
	}

	// Speed changes
	if p.Acceleration != 0 {
		p.Speed += p.Acceleration * dt
		if p.MaxSpeed > 0 && p.Speed > p.MaxSpeed {
			p.Speed = p.MaxSpeed
		}
		if p.Speed < 0 {
			p.Speed = 0
		}
	}

	// Angular velocity: rotate direction around Y axis
	if p.AngularVelocity != 0 {
		p.Direction = rotateVecY(p.Direction, p.AngularVelocity*dt)
	}

	// Linear motion
	p.Position = p.Position.Add(p.Direction.Scale(p.Speed * dt))
	p.Timer += dt
	if p.Timer >= p.Lifetime {
		p.Alive = false
	}
}

// rotateVecY rotates a vector around the Y axis by angle radians.
func rotateVecY(v Vec3, angle float32) Vec3 {
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	return Vec3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// ProjectileHitRadius is the collision radius for projectile hit checks.
// Matches the visual sphere radius (0.3) in enemy_projectile.tscn.
const ProjectileHitRadius float32 = 0.3

// PlayerHurtRadius is the player's hurtbox radius for projectile collision.
// Kept tight so bullet-hell patterns feel fair — players should be able to
// weave between projectiles that visually pass close by.
const PlayerHurtRadius float32 = 0.2
