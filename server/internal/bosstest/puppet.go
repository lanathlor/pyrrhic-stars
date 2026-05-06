package bosstest

import (
	"math"
	"math/rand/v2"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

// BotProfile defines the skill level of a simulated player.
type BotProfile string

const (
	ProfileSweaty  BotProfile = "sweaty"
	ProfileAverage BotProfile = "average"
	ProfileBad     BotProfile = "bad"
)

// ProfileParams holds tuning knobs that modulate puppet behavior quality.
type ProfileParams struct {
	// Reaction & awareness
	ReactionTime float32 // seconds before responding to telegraph
	SafetyMargin float32 // extra meters added to danger zone checks
	MechanicIQ   float32 // 0-1, probability of choosing correct dodge direction
	DodgeGreed   float32 // seconds melee puppets keep attacking before dodging (uptime optimization)

	// Rotation quality
	RotationDelay float32 // max random wasted time between casts (seconds)
	CooldownWaste float32 // probability of skipping an off-CD ability this tick
	DefensiveUse  float32 // probability of using defensives during telegraph

	// Movement (from class WalkSpeed)
	MoveSpeed float32
}

var profileTable = map[BotProfile]ProfileParams{
	ProfileSweaty:  {ReactionTime: 0.15, SafetyMargin: 1.5, MechanicIQ: 0.95, DodgeGreed: 0.70, RotationDelay: 0, CooldownWaste: 0.02, DefensiveUse: 0.9},
	ProfileAverage: {ReactionTime: 0.40, SafetyMargin: 0.5, MechanicIQ: 0.60, DodgeGreed: 0.35, RotationDelay: 0.15, CooldownWaste: 0.20, DefensiveUse: 0.5},
	ProfileBad:     {ReactionTime: 1.00, SafetyMargin: 0.0, MechanicIQ: 0.20, DodgeGreed: 0, RotationDelay: 0.30, CooldownWaste: 0.40, DefensiveUse: 0.1},
}

// Preferred range per class (how far the puppet wants to stand from the boss).
var classPreferredRange = map[string]float32{
	entity.ClassGunner:      10.0,
	entity.ClassVanguard:    2.5,
	entity.ClassBladeDancer: 3.0,
}

// classSafetyScale reduces SafetyMargin for melee classes.
// Melee classes need 0 margin — dodging happens at exact danger boundary.
// SafetyMargin only helps ranged classes (wider buffer at no DPS cost).
var classSafetyScale = map[string]float32{
	entity.ClassGunner:      1.0,
	entity.ClassVanguard:    0.3,
	entity.ClassBladeDancer: 0.3,
}

// PuppetContext is the tick context passed to puppet BT leaves.
type PuppetContext struct {
	Puppet     *PlayerPuppet
	World      *system.World
	Boss       *entity.Enemy
	BossDef    *enemyai.EnemyDef
	ActiveAbil *ability.AbilityDef // resolved with phase overrides, nil if none
	AllPuppets []*PlayerPuppet
	Dt         float32
}

// PlayerPuppet is a BT-driven player simulation for fuzz tests.
type PlayerPuppet struct {
	Player  *entity.Player
	Profile BotProfile
	Params  ProfileParams
	Rng     *rand.Rand

	tree           *bt.Tree
	preferredRange float32

	// State tracking for reaction timing
	lastBossState       entity.EnemyState
	telegraphElapsed    float32 // time since last telegraph onset
	dodgedThisTelegraph bool    // already dodged during current telegraph (prevents continuous dodging)
	rotationWait        float32 // artificial delay before next cast
}

// NewPuppet creates a player puppet with the given profile and seeded RNG.
// If reg is non-nil, it attempts to resolve a YAML-defined behavior tree
// for the (class, boss, profile) triple before falling back to the hardcoded Go tree.
func NewPuppet(id uint16, class string, profile BotProfile, seed uint64, boss string, reg *PuppetTreeRegistry) *PlayerPuppet {
	p := entity.NewPlayer(id, class)
	p.Health = p.MaxHealth
	p.Alive = true
	p.Position = entity.Vec3{X: float32(id)*2 - 4, Y: 0.1, Z: 5}

	params := profileTable[profile]
	if scale, ok := classSafetyScale[class]; ok {
		params.SafetyMargin *= scale
	}
	// Ranged classes don't need DodgeGreed — they lose nothing by dodging early
	if classPreferredRange[class] > 5.0 {
		params.DodgeGreed = 0
	}
	if params.MoveSpeed == 0 {
		if cd, ok := entity.Classes[class]; ok {
			params.MoveSpeed = cd.Movement.WalkSpeed
		} else {
			params.MoveSpeed = 5.0
		}
	}

	prefRange := classPreferredRange[class]
	if prefRange == 0 {
		prefRange = 6.0
	}

	pp := &PlayerPuppet{
		Player:         p,
		Profile:        profile,
		Params:         params,
		Rng:            rand.New(rand.NewPCG(seed, seed+1)),
		preferredRange: prefRange,
	}

	// YAML tree lookup first, then hardcoded Go fallback
	if res := reg.Resolve(class, boss, profile); res != nil {
		pp.tree = res.Tree
		if res.PreferredRange != nil {
			pp.preferredRange = *res.PreferredRange
		}
	} else {
		pp.tree = classTree(class)
	}
	return pp
}

// Boss room bounds — keeps puppets from drifting into the hallway through connector walls.
const (
	bossRoomMinX float32 = -18.0
	bossRoomMaxX float32 = 18.0
	bossRoomMinZ float32 = -13.0
	bossRoomMaxZ float32 = 10.0
)

// Tick evaluates the puppet's behavior tree for one simulation tick.
// The BT generates movement and ability inputs that are fed into the system pipeline.
func (pp *PlayerPuppet) Tick(ctx *PuppetContext) {
	if !pp.Player.Alive {
		return
	}

	// Track telegraph onset for reaction timing
	pp.updateTelegraphTracking(ctx)

	// Face the boss (required for hitscan/melee hit resolution)
	pp.faceBoss(ctx.Boss)

	// Evaluate BT — leaves push inputs onto World.InputQueue
	pp.tree.Tick(ctx)

	// Clamp to boss room bounds (prevents drifting into hallway)
	pp.clampToBossRoom()

	// Emit position input so InputSystem applies clamping and obstacle push
	pp.emitPositionInput(ctx)
}

// clampToBossRoom constrains the puppet's position to the boss room.
func (pp *PlayerPuppet) clampToBossRoom() {
	p := &pp.Player.Position
	if p.X < bossRoomMinX {
		p.X = bossRoomMinX
	}
	if p.X > bossRoomMaxX {
		p.X = bossRoomMaxX
	}
	if p.Z < bossRoomMinZ {
		p.Z = bossRoomMinZ
	}
	if p.Z > bossRoomMaxZ {
		p.Z = bossRoomMaxZ
	}
}

// updateTelegraphTracking detects telegraph onset and tracks elapsed time.
// Preserves elapsed time during execute states so danger conditions remain active
// through the full attack lifecycle (telegraph → execute → damage resolution).
func (pp *PlayerPuppet) updateTelegraphTracking(ctx *PuppetContext) {
	currentState := ctx.Boss.State
	if isTelegraphState(currentState) {
		if !isTelegraphState(pp.lastBossState) && !isExecuteState(pp.lastBossState) {
			pp.telegraphElapsed = 0
			pp.dodgedThisTelegraph = false
		} else {
			pp.telegraphElapsed += ctx.Dt
		}
	} else if isExecuteState(currentState) {
		pp.telegraphElapsed += ctx.Dt
	} else {
		pp.telegraphElapsed = 0
		pp.dodgedThisTelegraph = false
	}
	pp.lastBossState = currentState
}

// faceBoss rotates the puppet to face the boss.
func (pp *PlayerPuppet) faceBoss(boss *entity.Enemy) {
	dx := boss.Position.X - pp.Player.Position.X
	dz := boss.Position.Z - pp.Player.Position.Z
	pp.Player.RotationY = float32(math.Atan2(float64(-dx), float64(-dz)))
	pp.Player.AimPitch = 0
}

// HasReacted returns true if enough time has passed since telegraph onset
// for this puppet's reaction time. Uses DodgeGreed for melee uptime optimization.
func (pp *PlayerPuppet) HasReacted() bool {
	threshold := pp.Params.ReactionTime
	if pp.Params.DodgeGreed > threshold {
		threshold = pp.Params.DodgeGreed
	}
	return pp.telegraphElapsed >= threshold
}

// HasReactedQuick returns true if ReactionTime has elapsed since telegraph onset.
// Ignores DodgeGreed — used for AoE and Charge avoidance where staying longer
// provides zero DPS benefit and only increases death risk.
func (pp *PlayerPuppet) HasReactedQuick() bool {
	return pp.telegraphElapsed >= pp.Params.ReactionTime
}

// DistToBoss returns the XZ distance from puppet to boss.
func (pp *PlayerPuppet) DistToBoss(boss *entity.Enemy) float32 {
	return pp.Player.Position.Flat().DistanceTo(boss.Position.Flat())
}

// emitPositionInput pushes the puppet's current position into the input queue.
// The InputSystem will clamp to level bounds and push out of obstacles.
func (pp *PlayerPuppet) emitPositionInput(ctx *PuppetContext) {
	pos := pp.Player.Position
	payload := codec.EncodePlayerInput(nil,
		pos.X, pos.Y, pos.Z,
		pp.Player.RotationY,
		ctx.World.TickNum,
		"run", 1.0, pp.Player.AimPitch,
	)
	ctx.World.InputQueue = append(ctx.World.InputQueue, system.InputMsg{
		PeerID:  pp.Player.ID,
		Opcode:  message.OpPlayerInput,
		Payload: payload,
	})
}

// MoveAwayFrom moves the puppet directly away from a position.
func (pp *PlayerPuppet) MoveAwayFrom(pos entity.Vec3, dt float32, speedMult float32) {
	dir := pp.Player.Position.Sub(pos).Flat()
	if dir.LengthSq() < 0.01 {
		angle := pp.Rng.Float32() * 2 * math.Pi
		dir = entity.Vec3{X: float32(math.Cos(float64(angle))), Z: float32(math.Sin(float64(angle)))}
	}
	dir = dir.Normalized()
	pp.Player.Position = pp.Player.Position.Add(dir.Scale(pp.Params.MoveSpeed * speedMult * dt))
}

// MoveToward moves the puppet toward a position.
func (pp *PlayerPuppet) MoveToward(pos entity.Vec3, dt float32) {
	dir := pos.Sub(pp.Player.Position).Flat()
	if dir.LengthSq() < 0.01 {
		return
	}
	dir = dir.Normalized()
	pp.Player.Position = pp.Player.Position.Add(dir.Scale(pp.Params.MoveSpeed * dt))
}

// MovePerpendicular moves the puppet perpendicular to a direction vector.
// MechanicIQ determines if the puppet picks the "better" side.
func (pp *PlayerPuppet) MovePerpendicular(threatDir entity.Vec3, dt float32) {
	perp := entity.Vec3{X: -threatDir.Z, Z: threatDir.X}

	toPlayer := pp.Player.Position.Flat()
	if pp.Rng.Float32() < pp.Params.MechanicIQ {
		if perp.Dot(toPlayer) < 0 {
			perp = perp.Scale(-1)
		}
	} else {
		if pp.Rng.Float32() < 0.5 {
			perp = perp.Scale(-1)
		}
	}

	perp = perp.Normalized()
	pp.Player.Position = pp.Player.Position.Add(perp.Scale(pp.Params.MoveSpeed * dt))
}

// TryCast pushes an ability input into the World's input queue.
// Returns true if the ability is expected to succeed (off cooldown, no GCD).
func (pp *PlayerPuppet) TryCast(ctx *PuppetContext, abilityID string) bool {
	// Rotation delay: simulate human hesitation between casts
	if pp.rotationWait > 0 {
		pp.rotationWait -= ctx.Dt
		return false
	}

	// Cooldown waste: sometimes skip casting even when available
	if pp.Params.CooldownWaste > 0 && pp.Rng.Float32() < pp.Params.CooldownWaste {
		return false
	}

	// Check if ability can succeed before queuing
	if pp.Player.GCDTimer > 0 {
		return false
	}
	if cd, ok := pp.Player.Cooldowns[abilityID]; ok && cd > 0 {
		return false
	}

	action, ok := abilityNameToAction(pp.Player, abilityID)
	if !ok {
		return false
	}

	payload := codec.EncodeAbilityInput(action, pp.Player.AimPitch, pp.Player.RotationY)
	ctx.World.InputQueue = append(ctx.World.InputQueue, system.InputMsg{
		PeerID:  pp.Player.ID,
		Opcode:  message.OpAbilityInput,
		Payload: payload,
	})

	// Roll rotation delay for next cast
	if pp.Params.RotationDelay > 0 {
		pp.rotationWait = pp.Rng.Float32() * pp.Params.RotationDelay
	}
	return true
}

// CanCast checks if an ability is ready (off cooldown and GCD clear).
func (pp *PlayerPuppet) CanCast(abilityID string) bool {
	p := pp.Player
	if p.GCDTimer > 0 {
		return false
	}
	if cd, ok := p.Cooldowns[abilityID]; ok && cd > 0 {
		return false
	}
	return true
}

// abilityNameToAction reverses the player's ActionMap to find the action byte for an ability.
func abilityNameToAction(p *entity.Player, abilityID string) (uint8, bool) {
	for action, id := range p.ActionMap {
		if id == abilityID {
			return action, true
		}
	}
	return 0, false
}

func isTelegraphState(s entity.EnemyState) bool {
	return s == entity.EnemyMeleeTelegraph ||
		s == entity.EnemyRangedTelegraph ||
		s == entity.EnemyAoETelegraph ||
		s == entity.EnemyChargeTelegraph
}

func isExecuteState(s entity.EnemyState) bool {
	return s == entity.EnemyMeleeAttack ||
		s == entity.EnemyRangedAttack ||
		s == entity.EnemyAoESlam ||
		s == entity.EnemyCharge
}
