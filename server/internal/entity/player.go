package entity

import "math"

// Class name constants.
const (
	ClassGunner      = "gunner"
	ClassVanguard    = "vanguard"
	ClassBladeDancer = "blade_dancer"
)

// PlayerState represents the state of a player character.
type PlayerState uint8

const (
	PlayerStateMove PlayerState = iota
	PlayerStateDodge
	PlayerStateAttack
	PlayerStateBlock
	PlayerStateStagger
	PlayerStateDead
)

// PlayerInput is the decoded position from a client for one frame.
// Client-authoritative movement: client sends its position, server validates.
type PlayerInput struct {
	PosX float32
	PosY float32
	PosZ float32
	RotY float32
	Tick uint32
}

// AbilityAction identifies a combat action sent via OpAbilityInput.
const (
	ActionShoot uint8 = 0 // gunner: fire weapon
	ActionMelee uint8 = 1 // vanguard/blade_dancer: light attack
	ActionHeavy uint8 = 2 // vanguard/blade_dancer: heavy attack
	ActionDodge uint8 = 3 // any class: dodge roll
	ActionGuard uint8 = 4 // blade_dancer: guard / barrier

	// Gunner abilities
	ActionOverclock        uint8 = 10
	ActionRechamber        uint8 = 11
	ActionRechamberConfirm uint8 = 12

	// Vanguard AoE abilities
	ActionBladeSwirl uint8 = 20
	ActionGroundSlam uint8 = 21

	// Blade Dancer: action IDs 30-49 encode (origin_config * 4 + dest_slot)
	ActionBDSpellBase uint8 = 30
)

// Blade Dancer configuration IDs.
const (
	ConfigOrbit   = 0
	ConfigFan     = 1
	ConfigLance   = 2
	ConfigScatter = 3
	ConfigCrown   = 4
)

// Player represents a player entity on the server.
type Player struct {
	PeerID    uint16
	Username  string // display name
	ClassName string // "gunner", "vanguard", "blade_dancer"

	// Spatial
	Position  Vec3
	RotationY float32
	AimPitch  float32 // for gunner hitscan
	Velocity  Vec3
	OnGround  bool

	// State
	Health         float32
	MaxHealth      float32
	State          PlayerState
	Alive          bool
	InCombat       bool   // true when targeted by an enemy or recently damaged
	LastDamageTick uint32 // tick when last took damage (for combat exit timer)
	SpawnTick      uint32 // tick when player was spawned (reject stale positions)

	// Dodge
	IsRolling     bool
	RollTimer     float32
	RollCooldown  float32
	RollDirection Vec3

	// Invincibility (dodge i-frames)
	Invincible      bool
	InvincibleTimer float32

	// Gunner
	FireCooldown       float32
	OverclockActive    bool
	OverclockTimer     float32 // remaining buff duration (7s)
	OverclockCooldown  float32 // remaining cooldown (15s)
	RechamberPhase     uint8   // 0=idle, 1=windup, 2=timing_window, 3=lockout
	RechamberTimer     float32
	RechamberBuff      bool
	RechamberBuffTimer float32 // remaining damage buff duration (4s)

	// Vanguard
	Stamina            float32
	MaxStamina         float32
	StaminaRegen       float32
	StaminaDelay       float32
	ComboStep          int
	IsBlocking         bool
	ParryTimer         float32
	BladeSwirl         bool
	BladeSwirlTimer    float32 // remaining spin duration (1.5s)
	BladeSwirlTicks    int     // damage ticks delivered so far
	BladeSwirlCooldown float32 // remaining cooldown (10s)
	GroundSlamCooldown float32 // remaining cooldown (8s)

	// Blade dancer
	Config      int // 0=orbit, 1=fan, 2=lance, 3=scatter, 4=crown
	GCDTimer    float32
	GuardActive bool
	GuardTimer  float32
	BDShieldHP  float32 // temporary shield from Orbit-destination spells
	BDDRFactor  float32 // active damage reduction multiplier (0 = none)
	BDDRTimer   float32 // remaining DR buff duration

	// Animation (forwarded to clients)
	AnimName  string
	AnimSpeed float32

	// Input
	LastInput PlayerInput

	// Lobby
	Ready bool
}

// NewPlayer creates a player with class defaults.
func NewPlayer(peerID uint16, className string) *Player {
	p := &Player{
		PeerID:    peerID,
		ClassName: className,
		Alive:     true,
		OnGround:  true,
		AnimName:  "idle",
		AnimSpeed: 1.0,
	}
	switch className {
	case ClassVanguard:
		p.MaxHealth = 200.0
		p.Stamina = 100.0
		p.MaxStamina = 100.0
		p.StaminaRegen = 30.0
		p.StaminaDelay = 0.6
	default:
		p.MaxHealth = 150.0
	}
	p.Health = p.MaxHealth // spawn at full HP
	return p
}

// NewPlayerNoPTR creates a player with class defaults (value type).
func NewPlayerNoPTR(peerID uint16, className string) Player {
	p := Player{
		PeerID:    peerID,
		ClassName: className,
		Alive:     true,
		OnGround:  true,
		AnimName:  "idle",
		AnimSpeed: 1.0,
	}
	switch className {
	case ClassVanguard:
		p.MaxHealth = 200.0
		p.Stamina = 100.0
		p.MaxStamina = 100.0
		p.StaminaRegen = 30.0
		p.StaminaDelay = 0.6
	default:
		p.MaxHealth = 150.0
	}
	p.Health = p.MaxHealth // spawn at full HP
	return p
}

// classStats holds per-class movement tuning.
type classStats struct {
	WalkSpeed   float32
	SprintSpeed float32
	JumpVel     float32
	GroundAccel float32
	GroundDecel float32
	AirAccel    float32
	AirDecel    float32
	RollSpeed   float32
	RollDur     float32
	RollCD      float32
}

var classStatsTable = map[string]classStats{
	ClassGunner: {
		WalkSpeed: 5.5, SprintSpeed: 7.7, JumpVel: 4.0,
		GroundAccel: 25.0, GroundDecel: 18.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 14.0, RollDur: 0.3, RollCD: 2.5,
	},
	ClassVanguard: {
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.4, RollCD: 1.0,
	},
	ClassBladeDancer: {
		WalkSpeed: 6.0, SprintSpeed: 9.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 15.0, RollDur: 0.2, RollCD: 0.5,
	},
}

func (p *Player) stats() classStats {
	s, ok := classStatsTable[p.ClassName]
	if !ok {
		return classStatsTable[ClassGunner]
	}
	return s
}

// Note: Player movement is client-authoritative. The server stores positions
// received from the client. ProcessMovement/startRoll/updateAnimation are removed.

// ApplyDamage reduces health considering class-specific defenses.
func (p *Player) ApplyDamage(amount float32) float32 {
	if p.State == PlayerStateDead || !p.Alive {
		return 0
	}
	if p.Invincible {
		return 0
	}
	// Vanguard parry
	if p.ClassName == ClassVanguard && p.IsBlocking && p.ParryTimer > 0 {
		return 0
	}
	// Vanguard block
	if p.ClassName == ClassVanguard && p.IsBlocking {
		amount *= 0.3
	}
	// Vanguard blade swirl — 20% DR while spinning
	if p.ClassName == ClassVanguard && p.BladeSwirl {
		amount *= 0.8
	}
	// Blade dancer guard
	if p.ClassName == ClassBladeDancer && p.GuardActive {
		amount *= 0.5
	}
	// Blade dancer DR buff (from transition spells)
	if p.BDDRFactor > 0 && p.BDDRTimer > 0 {
		amount *= p.BDDRFactor
	}
	// Blade dancer shield absorb
	if p.BDShieldHP > 0 {
		if amount <= p.BDShieldHP {
			p.BDShieldHP -= amount
			return amount // damage absorbed, health unchanged
		}
		amount -= p.BDShieldHP
		p.BDShieldHP = 0
	}
	p.Health -= amount
	if p.Health <= 0 {
		p.Health = 0
		p.State = PlayerStateDead
		p.Alive = false
	}
	return amount
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// Forward returns the unit vector in the direction the player is facing (Godot convention: -Z is forward).
func (p *Player) Forward() Vec3 {
	s := float32(math.Sin(float64(p.RotationY)))
	c := float32(math.Cos(float64(p.RotationY)))
	return Vec3{-s, 0, -c}
}

// EyePosition returns the position of the player's eyes (for hitscan).
func (p *Player) EyePosition() Vec3 {
	return p.Position.Add(Vec3{0, 1.6, 0})
}

// AimDirection returns the direction the player is aiming (yaw + pitch).
// For FPS (gunner), pitch is sent. For 3rd person, pitch is 0 (aim forward).
func (p *Player) AimDirection() Vec3 {
	// Use RotationY for yaw, AimPitch for pitch
	pitch := p.AimPitch
	yaw := p.RotationY
	cp := float32(math.Cos(float64(pitch)))
	sp := float32(math.Sin(float64(pitch)))
	sy := float32(math.Sin(float64(yaw)))
	cy := float32(math.Cos(float64(yaw)))
	return Vec3{-sy * cp, sp, -cy * cp}
}
