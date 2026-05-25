package ability

import (
	"math"
	"math/rand"

	"codex-online/server/internal/entity"
)

// Assault tuning constants.
const (
	assaultMagSize          = 30
	assaultTacticalReload   = 1.5  // seconds, rounds remaining
	assaultEmptyReload      = 2.2  // seconds, magazine dry
	assaultStabilityDecay   = 0.08 // per shot
	assaultStabilityRate    = 2.0  // recovery per second
	assaultStabilityDelay   = 0.15 // seconds before recovery starts
	assaultMaxSpreadDeg     = 3.0  // degrees at stability 0
	assaultOverclockRecov   = 1.5  // multiplier on recovery rate
	assaultPressureMax      = 10
	assaultPressureTimeout  = 2.0  // seconds
	assaultPressureBonus    = 0.03 // per stack, fraction of base damage
	assaultEnhancedBatch    = 5    // rounds generated at max pressure
	assaultEnhancedMaxRes   = 10   // max reserve
	assaultEnhancedBase     = 15.0 // base bonus damage per enhanced round
	assaultEnhancedPerStack = 1.5  // bonus per pressure stack
	assaultMagDumpCD        = 12.0
	assaultMagDumpRPS       = 2   // rounds per tick during mag dump
	assaultMagDumpStab      = 0.8 // locked stability during mag dump

	// Steadiness: movement-based accuracy penalty (separate from Stability)
	assaultSteadinessDecay    = 0.12 // steadiness lost per unit of horizontal speed
	assaultSteadinessRecov    = 2.5  // recovery per second when stationary
	assaultSteadinessDelay    = 0.2  // seconds of no movement before recovery starts
	assaultMaxSteadinessDeg   = 1.5  // max additional spread degrees from low steadiness
	assaultSteadinessSpeedMin = 0.5  // speed below this counts as stationary
)

var assaultMaxSpreadRad = float32(assaultMaxSpreadDeg * math.Pi / 180.0)
var assaultMaxSteadinessRad = float32(assaultMaxSteadinessDeg * math.Pi / 180.0)

// GunnerAssaultState tracks all per-player Assault spec state.
type GunnerAssaultState struct {
	// Magazine
	MagCurrent  int
	MagMax      int
	Reloading   bool
	ReloadTimer float32
	ReloadTotal float32

	// Stability
	Stability      float32
	StabilityTimer float32

	// Pressure
	PressureStacks       uint8
	PressureTarget       uint16
	PressureTimer        float32
	OverclockPressureOdd bool

	// Enhanced Rounds
	EnhancedReserve int
	EnhancedLoaded  int

	// Mag Dump
	MagDumpActive    bool
	MagDumpTimer     float32
	MagDumpShotsLeft int

	// Steadiness (movement-based accuracy)
	Steadiness      float32
	SteadinessTimer float32
	PrevPos         entity.Vec3
	prevPosInit     bool

	// Track whether we already generated a batch at max stacks
	pressureWasMax bool
}

// GunnerWireState returns quantized fields for the wire format.
// Satisfies the codec interface without an import cycle.
func (s *GunnerAssaultState) GunnerWireState() (mag, magMax, stab, steadiness, pressure, enhanced, flags uint8) {
	mag = uint8(s.MagCurrent)
	magMax = uint8(s.MagMax)
	stab = uint8(entity.Clamp(s.Stability, 0, 1) * 255)
	steadiness = uint8(entity.Clamp(s.Steadiness, 0, 1) * 255)
	pressure = s.PressureStacks
	enhanced = uint8(s.EnhancedLoaded)
	if s.Reloading {
		flags |= 0x01
	}
	if s.MagDumpActive {
		flags |= 0x02
	}
	return
}

func getGunnerAssaultState(p *entity.Player) *GunnerAssaultState {
	if s, ok := p.AbilityState["gunner_assault"].(*GunnerAssaultState); ok {
		return s
	}
	s := &GunnerAssaultState{
		MagCurrent: assaultMagSize,
		MagMax:     assaultMagSize,
		Stability:  1.0,
		Steadiness: 1.0,
	}
	p.AbilityState["gunner_assault"] = s
	return s
}

// ---------------------------------------------------------------------------
// fire_shot handler
// ---------------------------------------------------------------------------

func fireShotAssaultHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getGunnerAssaultState(p)

	// Validate
	if state.Reloading {
		return CommitResult{Reason: "reloading"}
	}
	if state.MagDumpActive {
		return CommitResult{Reason: "mag dump active"}
	}
	if state.MagCurrent <= 0 && state.EnhancedLoaded <= 0 {
		return CommitResult{Reason: "empty magazine"}
	}

	// Consume ammo
	usingEnhanced := false
	if state.EnhancedLoaded > 0 {
		state.EnhancedLoaded--
		usingEnhanced = true
	} else {
		state.MagCurrent--
	}

	// Resolve hitscan with stability cone
	events := fireHitscan(eng, p, state, ctx)

	// Pressure tracking
	updatePressure(p, state, events)

	// Enhanced round bonus damage
	if usingEnhanced && len(events) > 0 {
		applyEnhancedBonus(p, state, events)
	}

	// Stability decay
	state.Stability = entity.Clamp(state.Stability-assaultStabilityDecay, 0, 1)
	state.StabilityTimer = 0

	// Set attack state (handler returns before engine sets this)
	p.State = entity.PlayerStateAttack

	// Cooldown (affected by Overclock BuffCooldownMult)
	cd := fireShotDef.Cooldown
	for i := range p.Buffs {
		if p.Buffs[i].Type == entity.BuffCooldownMult {
			cd *= p.Buffs[i].Value
		}
	}
	p.Cooldowns["fire_shot"] = cd
	p.State = entity.PlayerStateAttack

	// Sync munitions resource for wire format (reserve count)
	syncMunitionsResource(p, state)

	// Auto-reload on empty
	if state.MagCurrent <= 0 && state.EnhancedLoaded <= 0 && !state.Reloading {
		startReload(p, state)
	}

	return CommitResult{OK: true, Events: events}
}

// fireHitscan resolves a single hitscan shot with stability + steadiness cone offset.
func fireHitscan(eng *Engine, p *entity.Player, state *GunnerAssaultState, ctx *CommitContext) []DamageResult {
	origin := p.CommitterEyePos()
	direction := p.CommitterAimDir()

	// Combined spread: stability (sustained fire) + steadiness (movement)
	stabilitySpread := assaultMaxSpreadRad * (1.0 - state.Stability)
	steadinessSpread := assaultMaxSteadinessRad * (1.0 - state.Steadiness)
	spreadRad := stabilitySpread + steadinessSpread
	if spreadRad > 0.0001 {
		direction = applySpreadCone(direction, spreadRad)
	}

	// CommitterDamageMult already includes BuffDamageMult (rechamber buff)
	damage := fireShotDef.BaseDamage * p.CommitterDamageMult()

	eng.hitBuf = eng.hitBuf[:0]
	return resolveHitscanDir(eng.hitBuf, origin, direction, ctx.Targets, ctx.Obstacles, damage, ctx.SourceType, p.CommitterID())
}

// applySpreadCone rotates a direction vector by a random angle within a cone.
func applySpreadCone(dir entity.Vec3, maxAngle float32) entity.Vec3 {
	// Random angle around the aim axis
	theta := rand.Float64() * 2 * math.Pi
	// Random magnitude within cone (uniform distribution on disc)
	r := float32(math.Sqrt(rand.Float64())) * maxAngle

	// Build perpendicular basis
	up := entity.Vec3{X: 0, Y: 1, Z: 0}
	if math.Abs(float64(dir.Dot(up))) > 0.99 {
		up = entity.Vec3{X: 1, Y: 0, Z: 0}
	}
	right := dir.Cross(up).Normalized()
	realUp := right.Cross(dir).Normalized()

	// Offset direction
	sinR := float32(math.Sin(float64(r)))
	cosR := float32(math.Cos(float64(r)))
	cosT := float32(math.Cos(theta))
	sinT := float32(math.Sin(theta))

	return dir.Scale(cosR).Add(right.Scale(sinR * cosT)).Add(realUp.Scale(sinR * sinT)).Normalized()
}

// updatePressure handles pressure stack tracking after a shot.
func updatePressure(p *entity.Player, state *GunnerAssaultState, events []DamageResult) {
	if len(events) == 0 {
		// Miss → reset
		state.PressureStacks = 0
		state.PressureTarget = 0
		state.pressureWasMax = false
		return
	}

	targetID := events[0].TargetID
	if targetID != state.PressureTarget {
		// Target swap → reset to 1
		state.PressureStacks = 1
		state.PressureTarget = targetID
		state.pressureWasMax = false
	} else {
		// Same target → stack
		gain := uint8(1)
		if p.HasBuff("overclock") {
			// 1.5x build: alternate between 1 and 2
			if state.OverclockPressureOdd {
				gain = 2
			}
			state.OverclockPressureOdd = !state.OverclockPressureOdd
		}
		state.PressureStacks += gain
		if state.PressureStacks > assaultPressureMax {
			state.PressureStacks = assaultPressureMax
		}
	}
	state.PressureTimer = assaultPressureTimeout

	// Apply pressure damage bonus
	if state.PressureStacks > 0 {
		bonus := fireShotDef.BaseDamage * float32(state.PressureStacks) * assaultPressureBonus
		bonus *= (1.0 + p.GearStats.Mastery/100.0)
		bonus *= p.CommitterDamageMult()
		if bonus > 0 {
			t := events[0].Target
			if t != nil && t.TargetAlive() {
				if dealt := t.TargetApplyDamage(bonus); dealt > 0 {
					events[0].Amount += dealt
				}
			}
		}
	}

	// Generate enhanced round batch on reaching max (only on transition)
	if state.PressureStacks >= assaultPressureMax && !state.pressureWasMax {
		state.pressureWasMax = true
		state.EnhancedReserve += assaultEnhancedBatch
		if state.EnhancedReserve > assaultEnhancedMaxRes {
			state.EnhancedReserve = assaultEnhancedMaxRes
		}
	}
	// Reset flag when stacks drop below max (handled by timeout/miss/swap above)
}

// applyEnhancedBonus adds enhanced round bonus damage to the first hit.
func applyEnhancedBonus(p *entity.Player, state *GunnerAssaultState, events []DamageResult) {
	bonus := float32(assaultEnhancedBase) + float32(assaultEnhancedPerStack)*float32(state.PressureStacks)
	bonus *= (1.0 + p.GearStats.Identity/100.0)
	bonus *= p.CommitterDamageMult()
	if bonus > 0 {
		t := events[0].Target
		if t != nil && t.TargetAlive() {
			if dealt := t.TargetApplyDamage(bonus); dealt > 0 {
				events[0].Amount += dealt
			}
		}
	}
}

// syncMunitionsResource keeps the munitions wire field in sync with enhanced reserve.
func syncMunitionsResource(p *entity.Player, state *GunnerAssaultState) {
	if r := p.Resources["munitions"]; r != nil {
		r.Current = float32(state.EnhancedReserve)
	}
}

// ---------------------------------------------------------------------------
// Reload handler
// ---------------------------------------------------------------------------

var reloadDef = AbilityDef{
	ID: "reload", Name: "Reload",
	Handler: "reload_assault",
}

func reloadAssaultHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getGunnerAssaultState(p)
	if state.Reloading {
		return CommitResult{Reason: "already reloading"}
	}
	if state.MagCurrent >= state.MagMax {
		return CommitResult{Reason: "magazine full"}
	}
	if state.MagDumpActive {
		return CommitResult{Reason: "mag dump active"}
	}
	startReload(p, state)
	return CommitResult{OK: true}
}

func startReload(p *entity.Player, state *GunnerAssaultState) {
	dur := float32(assaultTacticalReload)
	if state.MagCurrent <= 0 {
		dur = assaultEmptyReload
	}
	dur /= p.TempoMult()
	state.Reloading = true
	state.ReloadTimer = dur
	state.ReloadTotal = dur
	// Lock out fire_shot for the reload duration
	p.Cooldowns["fire_shot"] = dur
}

// ---------------------------------------------------------------------------
// Load Enhanced handler
// ---------------------------------------------------------------------------

var loadEnhancedDef = AbilityDef{
	ID: "load_enhanced", Name: "Load Enhanced",
	Handler: "load_enhanced_assault",
}

func loadEnhancedHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getGunnerAssaultState(p)
	if state.EnhancedReserve <= 0 {
		return CommitResult{Reason: "no enhanced rounds"}
	}
	if state.EnhancedLoaded > 0 {
		return CommitResult{Reason: "already loaded"}
	}
	state.EnhancedLoaded = state.EnhancedReserve
	state.EnhancedReserve = 0
	syncMunitionsResource(p, state)
	return CommitResult{OK: true}
}

// ---------------------------------------------------------------------------
// Mag Dump handler
// ---------------------------------------------------------------------------

var magDumpDef = AbilityDef{
	ID: "mag_dump", Name: "Mag Dump",
	Handler:  "mag_dump_assault",
	Cooldown: assaultMagDumpCD,
}

func magDumpHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getGunnerAssaultState(p)
	if state.Reloading {
		return CommitResult{Reason: "reloading"}
	}
	if state.MagDumpActive {
		return CommitResult{Reason: "already dumping"}
	}
	totalRounds := state.MagCurrent + state.EnhancedLoaded
	if totalRounds <= 0 {
		return CommitResult{Reason: "no ammo"}
	}
	if cd := p.Cooldowns["mag_dump"]; cd > 0 {
		return CommitResult{Reason: "cooldown"}
	}

	state.MagDumpActive = true
	state.MagDumpShotsLeft = totalRounds
	// Duration based on rounds: at 2 rounds/tick (50ms tick), 30 rounds = 15 ticks = 0.75s
	ticks := float32(totalRounds+assaultMagDumpRPS-1) / float32(assaultMagDumpRPS)
	dur := ticks * 0.05 // 50ms per tick
	state.MagDumpTimer = dur
	// Lock out other abilities
	p.GCDTimer = dur
	p.Cooldowns["mag_dump"] = assaultMagDumpCD
	return CommitResult{OK: true}
}

// ---------------------------------------------------------------------------
// Tick handler (registered as "gunner_assault")
// ---------------------------------------------------------------------------

func gunnerAssaultTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	if p.ClassID != entity.ClassGunner {
		return nil
	}
	state := getGunnerAssaultState(p)

	tickGunnerReload(state, dt, p.TempoMult())
	tickGunnerStability(state, p, dt)
	tickGunnerPressure(state, dt)
	tickGunnerSteadiness(state, p, dt)

	var allEvents []DamageResult
	if state.MagDumpActive {
		allEvents = magDumpTick(eng, p, state, ctx)
	}

	syncMunitionsResource(p, state)
	return allEvents
}

// tickGunnerReload decrements the reload timer and refills the magazine when complete.
func tickGunnerReload(state *GunnerAssaultState, dt float32, tempoMult float32) {
	if !state.Reloading {
		return
	}
	state.ReloadTimer -= dt * tempoMult
	if state.ReloadTimer <= 0 {
		state.MagCurrent = state.MagMax
		state.Reloading = false
		state.ReloadTimer = 0
		state.ReloadTotal = 0
	}
}

// tickGunnerStability advances the stability recovery timer and recovers stability.
func tickGunnerStability(state *GunnerAssaultState, p *entity.Player, dt float32) {
	state.StabilityTimer += dt
	if state.StabilityTimer > assaultStabilityDelay && state.Stability < 1.0 {
		rate := float32(assaultStabilityRate)
		if p.HasBuff("overclock") {
			rate *= assaultOverclockRecov
		}
		state.Stability += rate * dt
		if state.Stability > 1.0 {
			state.Stability = 1.0
		}
	}
}

// tickGunnerPressure times out pressure stacks when the player stops hitting the same target.
func tickGunnerPressure(state *GunnerAssaultState, dt float32) {
	if state.PressureStacks == 0 {
		return
	}
	state.PressureTimer -= dt
	if state.PressureTimer <= 0 {
		state.PressureStacks = 0
		state.PressureTarget = 0
		state.PressureTimer = 0
		state.pressureWasMax = false
	}
}

// tickGunnerSteadiness tracks movement and decays or recovers steadiness accordingly.
func tickGunnerSteadiness(state *GunnerAssaultState, p *entity.Player, dt float32) {
	if !state.prevPosInit {
		state.PrevPos = p.Position
		state.prevPosInit = true
	}
	dx := p.Position.X - state.PrevPos.X
	dz := p.Position.Z - state.PrevPos.Z
	horizSpeed := float32(math.Sqrt(float64(dx*dx+dz*dz))) / dt
	state.PrevPos = p.Position

	if horizSpeed > assaultSteadinessSpeedMin {
		state.Steadiness -= assaultSteadinessDecay * horizSpeed * dt
		if state.Steadiness < 0 {
			state.Steadiness = 0
		}
		state.SteadinessTimer = 0
	} else {
		state.SteadinessTimer += dt
		if state.SteadinessTimer > assaultSteadinessDelay && state.Steadiness < 1.0 {
			state.Steadiness += assaultSteadinessRecov * dt
			if state.Steadiness > 1.0 {
				state.Steadiness = 1.0
			}
		}
	}
}

// magDumpTick fires multiple rounds per tick during mag dump.
func magDumpTick(eng *Engine, p *entity.Player, state *GunnerAssaultState, ctx *TickContext) []DamageResult {
	var events []DamageResult

	roundsThisTick := min(assaultMagDumpRPS, state.MagDumpShotsLeft)

	// Lock stability during dump
	origStability := state.Stability
	state.Stability = assaultMagDumpStab

	for range roundsThisTick {
		// Consume ammo
		usingEnhanced := false
		if state.EnhancedLoaded > 0 { //nolint:gocritic // ifElseChain: break targets the for loop, switch would change semantics
			state.EnhancedLoaded--
			usingEnhanced = true
		} else if state.MagCurrent > 0 {
			state.MagCurrent--
		} else {
			break
		}

		// Resolve hitscan
		commitCtx := &CommitContext{
			Committer:  p,
			Targets:    ctx.Targets,
			Obstacles:  ctx.Obstacles,
			SourceType: 0,
		}
		hits := fireHitscan(eng, p, state, commitCtx)

		// Pressure tracking
		updatePressure(p, state, hits)

		// Enhanced bonus
		if usingEnhanced && len(hits) > 0 {
			applyEnhancedBonus(p, state, hits)
		}

		events = append(events, hits...)
		state.MagDumpShotsLeft--
	}

	// Restore stability (will decay naturally next shot)
	state.Stability = origStability

	// Check if dump is complete
	if state.MagDumpShotsLeft <= 0 {
		state.MagDumpActive = false
		state.MagDumpTimer = 0
		state.MagCurrent = 0
		state.EnhancedLoaded = 0
		// Force empty reload
		startReload(p, state)
	}

	return events
}
