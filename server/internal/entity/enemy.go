package entity

import "slices"

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
	EnemyPatrol
)

// Enemy represents a server-side enemy entity (e.g. a boss).
type Enemy struct {
	Combatant

	Phase             int
	PhaseTransitioned []int

	State         EnemyState
	StateTimer    float32
	ChaseTimer    float32
	LastAttack    string
	ActiveAbility int // index into EnemyDef.Abilities

	// SecsSinceDealtDamage tracks how long since this enemy last landed any
	// damage on a player (maintained by the combat system, includes projectiles).
	// Read by AI to detect a boss whose abilities keep whiffing (the pillar
	// cheese: player commits the boss then ducks behind cover before it lands).
	SecsSinceDealtDamage float32

	// Target
	TargetPlayerID uint16

	// Melee (set by brain when starting melee ability)
	MeleeConeAngle float32 // full cone angle in radians (for client telegraph)
	MeleeRange     float32 // active melee ability range (for client telegraph)

	// Ranged
	RangedTargetPos Vec3

	// Charge
	ChargeDirection  Vec3
	ChargeDistance   float32
	ChargeHitPlayers []uint16

	// Threat table — tracks which players are engaged (peerID → threat)
	ThreatTable map[uint16]float32

	// Debuffs — active status effects applied by player abilities
	Debuffs []ActiveDebuff

	// Patrol — trash mobs patrol between two waypoints
	PatrolA      Vec3
	PatrolB      Vec3
	PatrolTarget int // 0 = heading to A, 1 = heading to B

	// Dungeon mob fields
	AggroRadius   float32 // distance at which mob detects players
	IsBoss        bool    // true for the boss, false for trash
	LeashOrigin   Vec3    // where the mob spawned (for leash behavior)
	LeashRadius   float32 // max distance from spawn before resetting
	DefName       string  // name of the EnemyDef, for client-side identification
	GroupID       int     // mobs with the same GroupID aggro together (0 = no group)
	BaseMaxHealth float32 // original MaxHealth before instance scaling (set at spawn)
}

// NewEnemy creates a fresh enemy with the given max health.
func NewEnemy(id uint16, maxHealth float32, defName string) *Enemy {
	return &Enemy{
		Combatant: Combatant{
			ID:        id,
			MaxHealth: maxHealth,
			Health:    maxHealth,
			Alive:     true,
		},
		BaseMaxHealth:     maxHealth,
		DefName:           defName,
		Phase:             1,
		PhaseTransitioned: []int{},
		State:             EnemyIdle,
		ChargeHitPlayers:  []uint16{},
		ThreatTable:       make(map[uint16]float32),
	}
}

// Reset restores the enemy to full health. initialState defaults to EnemyChase.
func (e *Enemy) Reset(spawnPos Vec3, initialState ...EnemyState) {
	e.Health = e.MaxHealth
	e.Phase = 1
	e.PhaseTransitioned = []int{}
	state := EnemyChase
	if len(initialState) > 0 {
		state = initialState[0]
	}
	e.State = state
	e.StateTimer = 0
	e.ChaseTimer = 0
	e.LastAttack = ""
	e.ActiveAbility = 0
	e.TargetPlayerID = 0
	e.Position = spawnPos
	e.RotationY = 0
	e.Velocity = Vec3{}
	e.Alive = true
	e.ChargeHitPlayers = []uint16{}
	e.ChargeDistance = 0
	e.ThreatTable = make(map[uint16]float32)
	e.Debuffs = e.Debuffs[:0]
}

// ApplyDamage reduces enemy health and checks phase transitions.
// Returns actual damage dealt.
func (e *Enemy) ApplyDamage(amount float32) (dealt float32, phaseTrigger int) {
	if e.State == EnemyDead {
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
	return slices.Contains(e.PhaseTransitioned, p)
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
	case EnemyRangedAttack, EnemyAoESlam:
		e.StateTimer = 0.1
	case EnemyAoETelegraph:
		e.StateTimer = e.getAoETelegraphTime()
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
	case EnemyPatrol:
		e.Velocity = Vec3{}
		e.ChaseTimer = 0
	}
}

// AddThreat increases a player's threat on this enemy.
func (e *Enemy) AddThreat(peerID uint16, amount float32) {
	if e.ThreatTable == nil {
		e.ThreatTable = make(map[uint16]float32)
	}
	e.ThreatTable[peerID] += amount
}

// HasThreat returns true if the player is on this enemy's threat table.
func (e *Enemy) HasThreat(peerID uint16) bool {
	_, ok := e.ThreatTable[peerID]
	return ok
}

// ClearThreat wipes the threat table.
func (e *Enemy) ClearThreat() {
	e.ThreatTable = make(map[uint16]float32)
}

// MeleeRange is the melee hit check distance.
const MeleeRange float32 = 3.0

// --- Committer interface (overrides for enemy-specific behavior) ---

func (e *Enemy) CommitterEyePos() Vec3        { return e.EyePos(1.5) }
func (e *Enemy) CommitterAimDir() Vec3        { return e.Forward() }
func (e *Enemy) CommitterAlive() bool         { return e.Alive && e.State != EnemyDead }
func (e *Enemy) CommitterDamageMult() float32 { return 1.0 }

// --- Target interface (overrides for enemy-specific behavior) ---

func (e *Enemy) TargetAlive() bool { return e.Alive && e.State != EnemyDead }
func (e *Enemy) TargetApplyDamage(a float32) float32 {
	if vuln := e.GetDebuffValue(DebuffVulnerability); vuln > 0 {
		a *= (1.0 + vuln)
	}
	dealt, _ := e.ApplyDamage(a)
	return dealt
}

// AddDebuff adds or replaces a debuff by ID.
func (e *Enemy) AddDebuff(d ActiveDebuff) {
	for i := range e.Debuffs {
		if e.Debuffs[i].ID == d.ID {
			e.Debuffs[i] = d
			return
		}
	}
	e.Debuffs = append(e.Debuffs, d)
}

// RemoveDebuff removes a debuff by ID.
func (e *Enemy) RemoveDebuff(id string) {
	for i := range e.Debuffs {
		if e.Debuffs[i].ID == id {
			e.Debuffs = slices.Delete(e.Debuffs, i, i+1)
			return
		}
	}
}

// HasDebuff returns true if any active debuff of the given type exists.
func (e *Enemy) HasDebuff(debuffType string) bool {
	for i := range e.Debuffs {
		if e.Debuffs[i].Type == debuffType {
			return true
		}
	}
	return false
}

// GetDebuffValue returns the strongest (highest) value for a debuff type, or 0.
func (e *Enemy) GetDebuffValue(debuffType string) float32 {
	var best float32
	for i := range e.Debuffs {
		if e.Debuffs[i].Type == debuffType && e.Debuffs[i].Value > best {
			best = e.Debuffs[i].Value
		}
	}
	return best
}

// TickDebuffs decrements durations and removes expired debuffs.
func (e *Enemy) TickDebuffs(dt float32) {
	alive := e.Debuffs[:0]
	for i := range e.Debuffs {
		e.Debuffs[i].Duration -= dt
		if e.Debuffs[i].Duration > 0 {
			alive = append(alive, e.Debuffs[i])
		}
	}
	e.Debuffs = alive
}
