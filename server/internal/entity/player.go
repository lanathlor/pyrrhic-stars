package entity

import "math"

// platingMinDamageFraction is the minimum fraction of original damage that
// always passes through Plating. Plating can never reduce a hit below this.
const platingMinDamageFraction float32 = 0.2

// DeliveryMethod identifies how a heal reaches its target.
// Harmony procs when consecutive heals on the same target use different methods.
type DeliveryMethod uint8

const (
	DeliveryNone   DeliveryMethod = 0
	DeliveryDirect DeliveryMethod = 1
	DeliveryBeam   DeliveryMethod = 2
	DeliveryZone   DeliveryMethod = 3
)

// HarmonyState tracks the last delivery method used on each ally target.
// Only initialized for Harmonist players.
type HarmonyState struct {
	LastDelivery map[uint16]DeliveryMethod
}

// ConfluenceState tracks the Arcanotechnicien class-wide Confluence mechanic.
// Each completed spell adds 1 stack (max 5). Per stack: +8% spell power.
// No cast for 4s triggers decay at 1 stack/sec. Interrupt drops all stacks.
type ConfluenceState struct {
	Stacks     int
	MaxStacks  int
	IdleTimer  float32
	DecayRate  float32
	DecayTimer float32
}

// SpellPowerMult returns the spell power multiplier from Confluence stacks.
func (c *ConfluenceState) SpellPowerMult() float32 {
	return 1.0 + float32(c.Stacks)*0.08
}

// OnSpellComplete adds a stack and resets the idle timer.
func (c *ConfluenceState) OnSpellComplete() {
	if c.Stacks < c.MaxStacks {
		c.Stacks++
	}
	c.IdleTimer = 0
	c.DecayTimer = 0
}

// OnInterrupt drops all stacks and resets timers.
func (c *ConfluenceState) OnInterrupt() {
	c.Stacks = 0
	c.IdleTimer = 0
	c.DecayTimer = 0
}

// Tick advances the idle and decay timers by dt seconds.
// Decay starts after 4s idle, removing 1 stack per second.
func (c *ConfluenceState) Tick(dt float32) {
	if c.Stacks == 0 {
		return
	}
	wasDecaying := c.IdleTimer >= 4.0
	c.IdleTimer += dt
	if c.IdleTimer >= 4.0 {
		// Only count time spent past the 4s threshold toward decay.
		decayDt := dt
		if !wasDecaying {
			decayDt = c.IdleTimer - 4.0
		}
		c.DecayTimer += decayDt
		for c.DecayTimer >= 1.0 && c.Stacks > 0 {
			c.DecayTimer -= 1.0
			c.Stacks--
		}
	}
}

// GearStats holds aggregated stat bonuses from equipped gear.
type GearStats struct {
	Hull     float32
	Output   float32
	Plating  float32
	Tempo    float32
	Identity float32
	Mastery  float32
}

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
	Username   string    // display name
	ClassID    string    // "gunner", "vanguard", "blade_dancer"
	SpecID     string    // "assault", "blade", "multi_blade", etc.
	GearStats  GearStats // aggregated stat bonuses from equipped gear
	BaseHealth float32   // spec base HP (from SpecDef, before gear)

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

	// Channel state (player ability runner)
	ChannelAbilityID string
	ChannelTimer     float32
	ChannelCharge    float32
	ChannelPhase     uint8 // 0=idle, 1=commit, 2=execute, 3=cooldown

	// Harmony state (Harmonist spec only)
	Harmony *HarmonyState

	// Confluence state (all Arcanotechnicien specs)
	Confluence *ConfluenceState

	// Blade Dancer config (visual state for client)
	Config int // 0=orbit, 1=fan, 2=lance, 3=scatter, 4=crown

	// Visual state (forwarded to clients, server never interprets)
	VisualState uint8

	// Input
	LastInput PlayerInput

	// Lobby
	Ready bool
}

// NewPlayer creates a player with class defaults (default spec).
func NewPlayer(peerID uint16, className string) *Player {
	classDef, ok := Classes[className]
	if !ok {
		classDef = Classes[ClassGunner]
		className = ClassGunner
	}
	return NewPlayerWithSpec(peerID, className, classDef.DefaultSpec)
}

// NewPlayerWithSpec creates a player with a specific specialization.
func NewPlayerWithSpec(peerID uint16, className, specID string) *Player {
	classDef, ok := Classes[className]
	if !ok {
		classDef = Classes[ClassGunner]
		className = ClassGunner
	}

	spec := classDef.GetSpec(specID)
	if spec == nil {
		spec = classDef.FirstSpec()
	}

	// Fall back to ClassDef root values if spec has no data (should not happen).
	maxHealth := spec.MaxHealth
	resources := spec.Resources
	actionMap := spec.ActionMap
	if maxHealth == 0 {
		maxHealth = classDef.MaxHealth
	}
	if resources == nil {
		resources = classDef.Resources
	}
	if actionMap == nil {
		actionMap = classDef.ActionMap
	}

	p := &Player{
		Combatant: Combatant{
			ID:    peerID,
			Alive: true,
		},
		ClassID:      className,
		SpecID:       spec.ID,
		BaseHealth:   maxHealth,
		OnGround:     true,
		Resources:    make(map[string]*Resource, len(resources)),
		Cooldowns:    make(map[string]float32),
		ActionMap:    actionMap,
		AbilityState: make(map[string]any),
	}

	for name, tmpl := range resources {
		p.Resources[name] = &Resource{
			Current:    tmpl.Initial,
			Max:        tmpl.Max,
			Regen:      tmpl.Regen,
			RegenDelay: tmpl.RegenDelay,
		}
	}

	if className == ClassArcanotechnicien {
		p.Confluence = &ConfluenceState{MaxStacks: 5, DecayRate: 1.0}
		if spec.ID == "harmonist" {
			p.Harmony = &HarmonyState{LastDelivery: make(map[uint16]DeliveryMethod)}
		}
	}

	p.RecalcStats()
	p.Health = p.MaxHealth

	return p
}

// NewPlayerNoPTR creates a player with class defaults (value type).
func NewPlayerNoPTR(peerID uint16, className string) Player {
	p := NewPlayer(peerID, className)
	return *p
}

// specResources returns the resource templates for the player's active spec,
// falling back to the class-level resources.
func (p *Player) specResources() map[string]ResourceTemplate {
	if cd, ok := Classes[p.ClassID]; ok {
		if s := cd.GetSpec(p.SpecID); s != nil && s.Resources != nil {
			return s.Resources
		}
		return cd.Resources
	}
	return nil
}

// RecalcStats recomputes derived stats from GearStats.
// Must be called whenever equipment changes.
func (p *Player) RecalcStats() {
	p.MaxHealth = p.BaseHealth + p.GearStats.Hull
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}

	res := p.specResources()
	identity := p.GearStats.Identity
	switch p.ClassID {
	case ClassVanguard:
		if r := p.Resources["stamina"]; r != nil {
			tmpl := res["stamina"]
			r.Max = tmpl.Max + identity
			r.Regen = tmpl.Regen * (1.0 + identity/100.0)
		}
	case ClassGunner:
		if r := p.Resources["munitions"]; r != nil {
			tmpl := res["munitions"]
			r.Max = tmpl.Max + identity*0.1
			r.Regen = tmpl.Regen * (1.0 + identity/100.0)
		}
	case ClassBladeDancer:
		if r := p.Resources["resonance"]; r != nil {
			tmpl := res["resonance"]
			r.Max = tmpl.Max + identity
			r.Regen = tmpl.Regen * (1.0 / (1.0 + identity/100.0))
		}
	case ClassArcanotechnicien:
		if r := p.Resources["flux"]; r != nil {
			tmpl := res["flux"]
			r.Max = tmpl.Max * (1.0 + identity/100.0)
			r.Regen = tmpl.Regen * (1.0 + identity/200.0)
		}
	}
}

// TenacityEfficiency returns the stamina cost multiplier from the Identity
// stat for vanguard. 0 Identity = 1.0 (no reduction), higher = cheaper costs.
func (p *Player) TenacityEfficiency() float32 {
	if p.ClassID != ClassVanguard {
		return 1.0
	}
	return 1.0 / (1.0 + p.GearStats.Identity/200.0)
}

// ClassName returns the class identifier.
func (p *Player) ClassName() string { return p.ClassID }

// SpecName returns the spec identifier.
func (p *Player) SpecName() string { return p.SpecID }

// Movement returns the spec movement stats.
func (p *Player) Movement() ClassMovement {
	if cd, ok := Classes[p.ClassID]; ok {
		if s := cd.GetSpec(p.SpecID); s != nil {
			return s.Movement
		}
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

	// Plating: flat damage reduction, min 20% of original damage passes through.
	original := amount
	amount -= p.GearStats.Plating
	if floor := original * platingMinDamageFraction; amount < floor {
		amount = floor
	}

	// Vanguard parry/block: check BEFORE DR application so parries see the
	// pre-DR amount (for reflect damage, Devotion generation, stamina drain).
	preDR := amount
	parried := false

	// Blade Parry: flag counter-swing pending (builds Onslaught instead of resetting it).
	if p.ClassID == ClassVanguard && p.SpecID != "shield" && p.HasBuff("vg_parry") {
		type parrySetter interface{ SetParryPending() }
		if s, ok := p.AbilityState["vg_block"]; ok {
			if ps, ok := s.(parrySetter); ok {
				ps.SetParryPending()
				parried = true
			}
		}
	}

	// Shield Guard Parry: zero stamina drain, reflect damage pending, bonus Devotion.
	if p.ClassID == ClassVanguard && p.SpecID == "shield" && p.HasBuff("vg_shield_parry") {
		type parryReflector interface{ SetParryReflectPending(dmg float32) }
		if s, ok := p.AbilityState["vg_shield_block"]; ok {
			if pr, ok := s.(parryReflector); ok {
				pr.SetParryReflectPending(preDR)
				// Bonus Devotion from parry (2x rate)
				type devotionAdder interface{ AddCharges(absorbed, mastery float32) }
				if d, ok := p.AbilityState["devotion"]; ok {
					if da, ok := d.(devotionAdder); ok {
						da.AddCharges(preDR*2.0, p.GearStats.Mastery)
					}
				}
				parried = true
			}
		}
	}

	// Shield Block (non-parry): drain stamina proportional to pre-DR damage, generate Devotion.
	if p.ClassID == ClassVanguard && p.SpecID == "shield" && p.HasBuff("vg_shield_block") && !parried {
		drainFraction := float32(0.65) // 65% of incoming damage → stamina drain (must match ability.ShieldStaminaDrainFraction)
		if p.HasBuff("brace") {
			drainFraction *= 0.2 // Brace reduces drain to 20% of normal
		}
		staminaDrain := preDR * drainFraction * p.TenacityEfficiency()
		if stamina := p.Resources["stamina"]; stamina != nil {
			stamina.Current -= staminaDrain
			if stamina.Current < 0 {
				stamina.Current = 0
			}
			stamina.DelayTimer = stamina.RegenDelay
		}
		// Devotion generation from blocked damage (decays over sustained block)
		type devotionAdder interface{ AddCharges(absorbed, mastery float32) }
		type devotionMulter interface{ GetDevotionMult() float32 }
		devMult := float32(1.0)
		if sb, ok := p.AbilityState["vg_shield_block"]; ok {
			if dm, ok := sb.(devotionMulter); ok {
				devMult = dm.GetDevotionMult()
			}
		}
		if d, ok := p.AbilityState["devotion"]; ok {
			if da, ok := d.(devotionAdder); ok {
				da.AddCharges(preDR*devMult, p.GearStats.Mastery)
			}
		}
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
		// BD Flow: reset chain on death.
		if p.ClassID == ClassBladeDancer {
			type resettable interface{ Reset() }
			if s, ok := p.AbilityState["flow"]; ok {
				if r, ok := s.(resettable); ok {
					r.Reset()
				}
			}
		}
	}

	// Vanguard Onslaught: reset stacks on damage taken (unless parried).
	// Shield spec doesn't use Onslaught.
	if p.ClassID == ClassVanguard && p.SpecID != "shield" && !parried {
		type resettable interface{ Reset() }
		if s, ok := p.AbilityState["onslaught"]; ok {
			if r, ok := s.(resettable); ok {
				r.Reset()
			}
		}
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

// TempoMult returns the speed multiplier from the Tempo gear stat.
// 0 Tempo = 1.0x, 100 Tempo = 2.0x.
func (p *Player) TempoMult() float32 {
	return 1.0 + p.GearStats.Tempo/100.0
}

func (p *Player) CasterDamageMult() float32 {
	m := p.DamageMult()
	// Output: additive percentage. 0 Output = 1.0x, 100 Output = 2.0x.
	m *= (1.0 + p.GearStats.Output/100.0)
	if p.GodMode {
		m *= 100
	}
	return m
}

// --- Target interface (overrides for player-specific behavior) ---

func (p *Player) TargetApplyDamage(a float32) float32 { return p.ApplyDamage(a) }
