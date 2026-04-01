package entity

// EnemyState represents the enemy FSM state.
type EnemyState uint8

const (
	EnemyIdle EnemyState = iota
	EnemyChase
	EnemyMeleeTelegraph
	EnemyMeleeAttack
	EnemyRangedTelegraph
	EnemyRangedAttack
	EnemyAoETelegraph
	EnemyAoESlam
	EnemyChargeTelegraph
	EnemyCharge
	EnemyCooldown
	EnemyPhaseTransition
	EnemyDead
)

// Enemy represents a server-side enemy entity (the arena boss).
type Enemy struct {
	ID        uint16
	Position  Vec3
	RotationY float32
	Velocity  Vec3

	Health    float32
	MaxHealth float32
	Phase     int
	PhaseTransitioned []int

	State      EnemyState
	StateTimer float32
	ChaseTimer float32
	LastAttack string

	// Target
	TargetPlayerID uint16

	// Ranged
	RangedTargetPos Vec3

	// Charge
	ChargeDirection  Vec3
	ChargeDistance    float32
	ChargeHitPlayers []uint16

	// Alive
	Alive bool
}

// NewEnemy creates a fresh boss enemy.
func NewEnemy(id uint16) *Enemy {
	return &Enemy{
		ID:                id,
		MaxHealth:         2000.0,
		Health:            2000.0,
		Phase:             1,
		PhaseTransitioned: []int{},
		State:             EnemyIdle,
		Alive:             true,
		ChargeHitPlayers:  []uint16{},
	}
}

// Reset restores the enemy to full health and initial state.
func (e *Enemy) Reset(spawnPos Vec3) {
	e.Health = e.MaxHealth
	e.Phase = 1
	e.PhaseTransitioned = []int{}
	e.State = EnemyChase
	e.StateTimer = 0
	e.ChaseTimer = 0
	e.LastAttack = ""
	e.Position = spawnPos
	e.RotationY = 0
	e.Velocity = Vec3{}
	e.Alive = true
	e.ChargeHitPlayers = []uint16{}
	e.ChargeDistance = 0
}

// ApplyDamage reduces enemy health and checks phase transitions.
// Returns actual damage dealt.
func (e *Enemy) ApplyDamage(amount float32) (dealt float32, phaseTrigger int) {
	if e.State == EnemyDead || e.State == EnemyPhaseTransition {
		return 0, 0
	}
	e.Health -= amount
	if e.Health < 0 {
		e.Health = 0
	}
	dealt = amount

	if e.Health <= 0 {
		e.State = EnemyDead
		e.Alive = false
		e.Velocity = Vec3{}
		return dealt, 0
	}

	// Phase transitions
	ratio := e.Health / e.MaxHealth
	if ratio <= 0.3 && !e.hasPhase(3) {
		e.enterPhase(3)
		return dealt, 3
	}
	if ratio <= 0.6 && !e.hasPhase(2) {
		e.enterPhase(2)
		return dealt, 2
	}
	return dealt, 0
}

func (e *Enemy) hasPhase(p int) bool {
	for _, pp := range e.PhaseTransitioned {
		if pp == p {
			return true
		}
	}
	return false
}

func (e *Enemy) enterPhase(p int) {
	e.Phase = p
	e.PhaseTransitioned = append(e.PhaseTransitioned, p)
	e.ChangeState(EnemyPhaseTransition)
}

// ChangeState transitions the enemy to a new FSM state, setting timers.
func (e *Enemy) ChangeState(s EnemyState) {
	e.State = s
	e.Velocity = Vec3{}

	switch s {
	case EnemyChase:
		e.ChaseTimer = 0
	case EnemyMeleeTelegraph:
		e.StateTimer = e.getMeleeTelegraphTime()
	case EnemyMeleeAttack:
		e.StateTimer = 0.3
	case EnemyRangedTelegraph:
		e.StateTimer = e.getRangedTelegraphTime()
	case EnemyRangedAttack:
		e.StateTimer = 0.1
	case EnemyAoETelegraph:
		e.StateTimer = e.getAoETelegraphTime()
	case EnemyAoESlam:
		e.StateTimer = 0.1
	case EnemyChargeTelegraph:
		e.StateTimer = e.getChargeTelegraphTime()
		e.ChargeDirection = Vec3{}
	case EnemyCharge:
		e.ChargeDistance = 0
		e.ChargeHitPlayers = []uint16{}
		if e.ChargeDirection.LengthSq() < 0.01 {
			// Fallback: charge forward
			e.ChargeDirection = Vec3{0, 0, -1}
		}
	case EnemyCooldown:
		e.StateTimer = e.getCooldownTime()
	case EnemyPhaseTransition:
		e.StateTimer = 1.5
	case EnemyDead:
		e.Velocity = Vec3{}
		e.Alive = false
	}
}

// MeleeRange is the melee hit check distance.
const MeleeRange float32 = 3.0
