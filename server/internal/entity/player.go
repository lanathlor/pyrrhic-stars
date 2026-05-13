package entity

import "math"

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
	ActionShoot     uint8 = 0 // gunner: fire weapon
	ActionMelee     uint8 = 1 // vanguard/blade_dancer: light attack
	ActionHeavy     uint8 = 2 // vanguard/blade_dancer: heavy attack
	ActionDodge     uint8 = 3 // any class: dodge roll
	ActionGuard     uint8 = 4 // blade_dancer: guard / barrier
	ActionBlockStop uint8 = 5 // vanguard: release block

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
	Combatant
	Username string // display name
	ClassID  string // "gunner", "vanguard", "blade_dancer"

	// Spatial (player-specific)
	AimPitch float32 // for gunner hitscan
	OnGround bool

	// State
	State          PlayerState
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
	GodMode         bool // dev mode: permanent invincibility

	// Generic resources (stamina, shield, etc.)
	Resources map[string]*Resource

	// Ability state
	Cooldowns    map[string]float32 // ability_id → remaining cooldown
	GCDTimer     float32            // global cooldown timer
	ActionMap    map[uint8]string   // wire action_id → ability_id
	AbilityState map[string]any     // ability_id → custom handler state

	// Active buffs
	Buffs []ActiveBuff

	// Active DoTs (sourced from this player onto enemies)
	DoTs []ActiveDoT

	// Blade Dancer config (visual state for client)
	Config int // 0=orbit, 1=fan, 2=lance, 3=scatter, 4=crown

	// Visual state (forwarded to clients, server never interprets)
	VisualState uint8

	// Input
	LastInput PlayerInput

	// Lobby
	Ready bool
}

// NewPlayer creates a player with class defaults.
func NewPlayer(peerID uint16, className string) *Player {
	classDef, ok := Classes[className]
	if !ok {
		classDef = Classes[ClassGunner]
		className = ClassGunner
	}

	p := &Player{
		Combatant: Combatant{
			ID:        peerID,
			MaxHealth: classDef.MaxHealth,
			Alive:     true,
		},
		ClassID:      className,
		OnGround:     true,
		Resources:    make(map[string]*Resource, len(classDef.Resources)),
		Cooldowns:    make(map[string]float32),
		ActionMap:    classDef.ActionMap,
		AbilityState: make(map[string]any),
	}
	p.Health = p.MaxHealth

	for name, tmpl := range classDef.Resources {
		p.Resources[name] = &Resource{
			Current:    tmpl.Initial,
			Max:        tmpl.Max,
			Regen:      tmpl.Regen,
			RegenDelay: tmpl.RegenDelay,
		}
	}

	return p
}

// NewPlayerNoPTR creates a player with class defaults (value type).
func NewPlayerNoPTR(peerID uint16, className string) Player {
	p := NewPlayer(peerID, className)
	return *p
}

// ClassName returns the class identifier.
func (p *Player) ClassName() string { return p.ClassID }

// Movement returns the class movement stats.
func (p *Player) Movement() ClassMovement {
	if cd, ok := Classes[p.ClassID]; ok {
		return cd.Movement
	}
	return Classes[ClassGunner].Movement
}

// HasBuff returns true if the player has an active buff with the given ID.
func (p *Player) HasBuff(id string) bool {
	for i := range p.Buffs {
		if p.Buffs[i].ID == id {
			return true
		}
	}
	return false
}

// GetBuffValue returns the value of the first buff matching the given type,
// or 1.0 if no buff of that type is active.
func (p *Player) GetBuffValue(buffType string) float32 {
	for i := range p.Buffs {
		if p.Buffs[i].Type == buffType {
			return p.Buffs[i].Value
		}
	}
	return 1.0
}

// GetBuff returns a pointer to the first buff with the given ID, or nil.
func (p *Player) GetBuff(id string) *ActiveBuff {
	for i := range p.Buffs {
		if p.Buffs[i].ID == id {
			return &p.Buffs[i]
		}
	}
	return nil
}

// DamageReduction returns the product of all active damage_reduction buff values.
func (p *Player) DamageReduction() float32 {
	mult := float32(1.0)
	for i := range p.Buffs {
		if p.Buffs[i].Type == BuffDamageReduction {
			mult *= p.Buffs[i].Value
		}
	}
	return mult
}

// DamageMult returns the product of all active damage_mult buff values.
func (p *Player) DamageMult() float32 {
	mult := float32(1.0)
	for i := range p.Buffs {
		if p.Buffs[i].Type == BuffDamageMult {
			mult *= p.Buffs[i].Value
		}
	}
	return mult
}

// GetResource returns the current value of a resource, or 0 if not present.
func (p *Player) GetResource(name string) float32 {
	if r, ok := p.Resources[name]; ok {
		return r.Current
	}
	return 0
}

// SpendResource deducts amount from a resource. Returns false if insufficient.
func (p *Player) SpendResource(name string, amount float32) bool {
	r, ok := p.Resources[name]
	if !ok {
		return amount <= 0
	}
	if r.Current < amount {
		return false
	}
	r.Current -= amount
	r.DelayTimer = r.RegenDelay
	return true
}

// AddBuff adds a buff to the player. If a buff with the same ID exists, it replaces it.
func (p *Player) AddBuff(b ActiveBuff) {
	for i := range p.Buffs {
		if p.Buffs[i].ID == b.ID {
			p.Buffs[i] = b
			return
		}
	}
	p.Buffs = append(p.Buffs, b)
}

// RemoveBuff removes a buff by ID.
func (p *Player) RemoveBuff(id string) {
	for i := range p.Buffs {
		if p.Buffs[i].ID == id {
			p.Buffs[i] = p.Buffs[len(p.Buffs)-1]
			p.Buffs = p.Buffs[:len(p.Buffs)-1]
			return
		}
	}
}

// ApplyDamage reduces health considering active buffs and shields.
func (p *Player) ApplyDamage(amount float32) float32 {
	if p.State == PlayerStateDead || !p.Alive {
		return 0
	}
	if p.Invincible || p.GodMode {
		return 0
	}

	// Apply all damage_reduction buffs
	amount *= p.DamageReduction()
	if amount <= 0 {
		return 0
	}

	// Shield absorb
	if shield, ok := p.Resources["shield"]; ok && shield.Current > 0 {
		if amount <= shield.Current {
			shield.Current -= amount
			return amount // fully absorbed
		}
		amount -= shield.Current
		shield.Current = 0
	}

	p.Health -= amount
	if p.Health <= 0 {
		p.Health = 0
		p.State = PlayerStateDead
		p.Alive = false
	}
	return amount
}

// GetAbilityPhase returns a uint8 phase from an ability state that implements
// the phaser interface { GetPhase() uint8 }. Returns 0 if not found.
func (p *Player) GetAbilityPhase(id string) uint8 {
	if s, ok := p.AbilityState[id]; ok {
		type phaser interface{ GetPhase() uint8 }
		if ph, ok := s.(phaser); ok {
			return ph.GetPhase()
		}
	}
	return 0
}

// EyePosition returns the position of the player's eyes (for hitscan).
func (p *Player) EyePosition() Vec3 {
	return p.EyePos(1.6)
}

// AimDirection returns the direction the player is aiming (yaw + pitch).
// For FPS (gunner), pitch is sent. For 3rd person, pitch is 0 (aim forward).
func (p *Player) AimDirection() Vec3 {
	pitch := p.AimPitch
	yaw := p.RotationY
	cp := float32(math.Cos(float64(pitch)))
	sp := float32(math.Sin(float64(pitch)))
	sy := float32(math.Sin(float64(yaw)))
	cy := float32(math.Cos(float64(yaw)))
	return Vec3{-sy * cp, sp, -cy * cp}
}

// --- Caster interface (overrides for player-specific behavior) ---

func (p *Player) CasterEyePos() Vec3 { return p.EyePosition() }
func (p *Player) CasterAimDir() Vec3 { return p.AimDirection() }
func (p *Player) CasterDamageMult() float32 {
	m := p.DamageMult()
	if p.GodMode {
		m *= 100
	}
	return m
}

// --- Target interface (overrides for player-specific behavior) ---

func (p *Player) TargetApplyDamage(a float32) float32 { return p.ApplyDamage(a) }
