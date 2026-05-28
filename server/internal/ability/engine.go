package ability

import (
	"context"
	"log/slog"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// nopHandler is a slog.Handler that discards all output with zero allocations.
type nopHandler struct{}

func (nopHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (nopHandler) Handle(context.Context, slog.Record) error { return nil }
func (nopHandler) WithAttrs([]slog.Attr) slog.Handler        { return nopHandler{} }
func (nopHandler) WithGroup(string) slog.Handler             { return nopHandler{} }

// HandlerFunc is a Go function that handles complex ability execution.
type HandlerFunc func(eng *Engine, ctx *CommitContext) CommitResult

// TickHandlerFunc is a Go function that handles per-tick logic for an ability.
type TickHandlerFunc func(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult

// CommitContext carries the state needed to resolve an ability.
type CommitContext struct {
	Committer  entity.Committer
	Targets    []entity.Target
	Obstacles  []combat.Obstacle
	SourceType uint8 // combat.SourcePlayerAttack, SourceEnemyMelee, etc.

	// Heal targeting
	Allies       map[uint16]*entity.Player // all players in zone (for heal targeting)
	TargetPeerID uint16                    // client-specified ally target

	// Zone spawning (set by the system layer to inject zones into the world)
	SpawnZone func(zone *entity.HealingZone)

	// Link spawning (set by the system layer to inject damage links into the world)
	SpawnLink func(link *entity.DamageLink)
}

// TickContext carries the state needed for per-tick ability updates.
type TickContext struct {
	Targets   []entity.Target
	Obstacles []combat.Obstacle
}

// HealResult is emitted by ability resolution when a heal is applied.
type HealResult struct {
	TargetID      uint16
	SourceID      uint16
	Amount        float32
	Overheal      float32
	HitPos        entity.Vec3
	SourceType    uint8
	HarmonyProc   bool
	HarmonyAmount float32
}

// CommitResult is returned by the engine after attempting to commit an ability.
type CommitResult struct {
	OK     bool
	Events []DamageResult
	Heals  []HealResult
	Reason string
}

// Engine validates and executes abilities.
type Engine struct {
	abilities    map[string]*AbilityDef
	handlers     map[string]HandlerFunc
	tickHandlers map[string]TickHandlerFunc
	logger       *slog.Logger
	logDebug     bool // cached: handler enables Debug level
	logInfo      bool // cached: handler enables Info level

	// hitBuf is a reusable scratch buffer for hit resolution results.
	// Valid only until the next Commit or TickPlayer call.
	hitBuf []DamageResult
	// tickBuf is a reusable scratch buffer for TickPlayer DoT events.
	// Valid only until the next TickPlayer call.
	tickBuf []DamageResult
}

// NewEngine creates an engine and registers all abilities and handlers.
// Pass nil for logger to discard all log output.
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.New(nopHandler{})
	}
	ctx := context.Background()
	eng := &Engine{
		abilities:    make(map[string]*AbilityDef),
		handlers:     make(map[string]HandlerFunc),
		tickHandlers: make(map[string]TickHandlerFunc),
		logger:       logger,
		logDebug:     logger.Handler().Enabled(ctx, slog.LevelDebug),
		logInfo:      logger.Handler().Enabled(ctx, slog.LevelInfo),
	}
	registerAbilities(eng)
	registerHandlers(eng)
	return eng
}

// Register adds an ability definition to the engine.
func (eng *Engine) Register(def *AbilityDef) {
	eng.abilities[def.ID] = def
}

// RegisterHandler registers a named Go handler for complex abilities.
func (eng *Engine) RegisterHandler(name string, fn HandlerFunc) {
	eng.handlers[name] = fn
}

// RegisterTickHandler registers a per-tick handler for an ability.
func (eng *Engine) RegisterTickHandler(name string, fn TickHandlerFunc) {
	eng.tickHandlers[name] = fn
}

// GetAbility returns the ability definition by ID, or nil.
func (eng *Engine) GetAbility(id string) *AbilityDef {
	return eng.abilities[id]
}

// Commit looks up an ability by ID and executes it.
// The returned CommitResult.Events slice is backed by an internal buffer and is
// only valid until the next Commit or TickPlayer call.
func (eng *Engine) Commit(abilityID string, ctx *CommitContext) CommitResult {
	def := eng.abilities[abilityID]
	if def == nil {
		eng.logger.Warn("ability.commit.rejected",
			"ability", abilityID,
			"reason", "unknown ability",
		)
		return CommitResult{Reason: "unknown ability"}
	}
	return eng.doCommit(abilityID, def, ctx)
}

// CommitDef executes an ability from a provided definition (not looked up by ID).
// Use this when the caller has a modified/resolved copy (e.g. enemy phase overrides).
func (eng *Engine) CommitDef(def *AbilityDef, ctx *CommitContext) CommitResult {
	return eng.doCommit(def.ID, def, ctx)
}

func (eng *Engine) doCommit(abilityID string, def *AbilityDef, ctx *CommitContext) CommitResult {
	eng.hitBuf = eng.hitBuf[:0]

	caster := ctx.Committer

	if !caster.CommitterAlive() {
		if eng.logDebug {
			eng.logger.Debug("ability.commit.rejected",
				"ability", abilityID, "id", caster.CommitterID(), "reason", "dead")
		}
		return CommitResult{Reason: "dead"}
	}

	// Player-specific validation
	p, isPlayer := caster.(*entity.Player)
	if isPlayer {
		if reason := eng.validatePlayerCommit(abilityID, def, p); reason != "" {
			return CommitResult{Reason: reason}
		}
	}

	// If this ability has a custom handler, delegate entirely
	if def.Handler != "" {
		return eng.dispatchHandler(abilityID, def, caster, ctx)
	}

	// Spend resources (player only)
	if isPlayer {
		spendCommitCosts(def, p)
	}

	// BD Resonance: check threshold for damage amplification (before hit)
	var resonanceAmpFactor float32
	if isPlayer {
		resonanceAmpFactor = bdResonanceFactor(p, def)
	}

	// Resolve hit
	eng.hitBuf = resolveHit(eng.hitBuf, def, caster, ctx.Targets, ctx.Obstacles, ctx.SourceType)

	// Splash damage: AoE around primary hit target (excluding primary)
	if def.SplashRadius > 0 && def.SplashDamageFraction > 0 && len(eng.hitBuf) > 0 {
		eng.hitBuf = applySplashDamage(eng.hitBuf, def, caster, ctx)
	}

	// BD Flow: track transition and apply damage bonus
	if isPlayer && p.ClassID == entity.ClassBladeDancer && def.OriginConfig >= 0 && def.DestConfig >= 0 {
		applyBDFlowBonus(eng.hitBuf, p, def)
	}

	events := eng.hitBuf

	// BD Resonance: apply bonus damage and consume charge
	if resonanceAmpFactor > 0 {
		applyBDResonanceBonus(eng.hitBuf, p, resonanceAmpFactor)
	}

	// Player-specific post-commit effects
	if isPlayer {
		applyPostCommitEffects(abilityID, def, p, events)
	}

	eng.logCommit(abilityID, caster.CommitterID(), events)
	return CommitResult{OK: true, Events: events}
}

// logCommit emits an Info log summarising a successful ability commit.
func (eng *Engine) logCommit(abilityID string, casterID uint16, events []DamageResult) {
	if !eng.logInfo {
		return
	}
	var totalDmg float32
	for _, ev := range events {
		totalDmg += ev.Amount
	}
	eng.logger.Info("ability.commit",
		"ability", abilityID, "id", casterID,
		"hits", len(events), "damage", totalDmg)
}

// validatePlayerCommit checks all player-specific preconditions before committing an ability.
// Returns the rejection reason string, or "" if valid.
func (eng *Engine) validatePlayerCommit(abilityID string, def *AbilityDef, p *entity.Player) string {
	if p.GCDTimer > 0 {
		if eng.logDebug {
			eng.logger.Debug("ability.commit.rejected",
				"ability", abilityID, "id", p.ID,
				"reason", "gcd", "gcd_remaining", p.GCDTimer)
		}
		return ReasonGCD
	}

	if cd, ok := p.Cooldowns[abilityID]; ok && cd > 0 {
		if eng.logDebug {
			eng.logger.Debug("ability.commit.rejected",
				"ability", abilityID, "id", p.ID,
				"reason", ReasonCooldown, "cd_remaining", cd)
		}
		return ReasonCooldown
	}

	// Cancel active Blade block when committing any other ability
	if abilityID != IDVgBlock && abilityID != IDVgBlockStop && p.HasBuff(IDVgBlock) {
		EndVgBlock(p)
	}
	// Cancel active Shield block (except Shield Bash and Brace which work during block)
	if abilityID != IDVgShieldBlock && abilityID != IDVgShieldBlockStop &&
		abilityID != "shield_bash" && abilityID != IDBrace &&
		p.HasBuff(IDVgShieldBlock) {
		EndVgShieldBlock(p)
	}

	if def.OriginConfig >= 0 && def.OriginConfig != p.Config {
		if eng.logDebug {
			eng.logger.Debug("ability.commit.rejected",
				"ability", abilityID, "id", p.ID,
				"reason", "wrong config", "need", def.OriginConfig, "have", p.Config)
		}
		return "wrong config"
	}

	return eng.validateCommitCosts(abilityID, def, p)
}

// validateCommitCosts checks that the player has sufficient resources for each cost.
// Returns the rejection reason string, or "" if sufficient.
func (eng *Engine) validateCommitCosts(abilityID string, def *AbilityDef, p *entity.Player) string {
	for _, cost := range def.Costs {
		// School-aware flux validation: check the specific school pool.
		if cost.Resource == entity.ResourceFlux && def.School != "" && p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
			scaledCost := cost.Amount * p.AffinityCostMult(def.School)
			pool := p.FluxCommit.GetPool(def.School)
			if pool == nil || pool.Current < scaledCost {
				have := float32(0)
				if pool != nil {
					have = pool.Current
				}
				if eng.logDebug {
					eng.logger.Debug("ability.commit.rejected",
						"ability", abilityID, "id", p.ID,
						"reason", "insufficient "+def.School+" flux", "need", scaledCost, "have", have)
				}
				return "insufficient " + def.School + " flux"
			}
			continue
		}
		r, ok := p.Resources[cost.Resource]
		if !ok {
			if eng.logDebug {
				eng.logger.Debug("ability.commit.rejected",
					"ability", abilityID, "id", p.ID,
					"reason", "insufficient "+cost.Resource, "need", cost.Amount, "have", 0)
			}
			return "insufficient " + cost.Resource
		}
		effectiveCost := cost.Amount
		if cost.Resource == entity.ResourceStamina {
			effectiveCost *= p.TenacityEfficiency()
		}
		if r.Current < effectiveCost {
			if eng.logDebug {
				eng.logger.Debug("ability.commit.rejected",
					"ability", abilityID, "id", p.ID,
					"reason", "insufficient "+cost.Resource, "need", effectiveCost, "have", r.Current)
			}
			return "insufficient " + cost.Resource
		}
	}
	return ""
}

// dispatchHandler routes an ability to its registered Go handler and logs the result.
func (eng *Engine) dispatchHandler(abilityID string, def *AbilityDef, caster entity.Committer, ctx *CommitContext) CommitResult {
	fn, ok := eng.handlers[def.Handler]
	if !ok {
		eng.logger.Warn("ability.commit.rejected",
			"ability", abilityID, "id", caster.CommitterID(),
			"reason", "handler not found", "handler", def.Handler)
		return CommitResult{Reason: "handler not found: " + def.Handler}
	}
	result := fn(eng, ctx)
	if eng.logInfo {
		if result.OK {
			var totalDmg float32
			for _, ev := range result.Events {
				totalDmg += ev.Amount
			}
			eng.logger.Info("ability.commit",
				"ability", abilityID, "id", caster.CommitterID(),
				"handler", def.Handler, "hits", len(result.Events), "damage", totalDmg)
		} else if eng.logDebug {
			eng.logger.Debug("ability.commit.rejected",
				"ability", abilityID, "id", caster.CommitterID(),
				"handler", def.Handler, "reason", result.Reason)
		}
	}
	return result
}

// spendCommitCosts deducts each resource cost from the player.
func spendCommitCosts(def *AbilityDef, p *entity.Player) {
	for _, cost := range def.Costs {
		if cost.Resource == entity.ResourceFlux && def.School != "" && p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
			p.SpendFluxBySchool(def.School, cost.Amount)
			continue
		}
		effectiveCost := cost.Amount
		if cost.Resource == entity.ResourceStamina {
			effectiveCost *= p.TenacityEfficiency()
		}
		p.SpendResource(cost.Resource, effectiveCost)
	}
}

// bdResonanceFactor returns the resonance damage amplification factor for a BD player,
// or 0 if the ability or player doesn't qualify.
func bdResonanceFactor(p *entity.Player, def *AbilityDef) float32 {
	if p.ClassID != entity.ClassBladeDancer || def.DestConfig < 0 {
		return 0
	}
	res := p.Resources["resonance"]
	if res == nil || res.Current < 50 {
		return 0
	}
	if res.Current >= 100 {
		return 0.5
	}
	return 0.25
}

// applySplashDamage adds AoE splash hits around the primary target and returns the extended buf.
func applySplashDamage(hitBuf []DamageResult, def *AbilityDef, caster entity.Committer, ctx *CommitContext) []DamageResult {
	primaryID := hitBuf[0].TargetID
	splashDmg := def.BaseDamage * caster.CommitterDamageMult() * def.SplashDamageFraction
	splashTargets := make([]entity.Target, 0, len(ctx.Targets))
	for _, t := range ctx.Targets {
		if t != nil && t.TargetID() != primaryID {
			splashTargets = append(splashTargets, t)
		}
	}
	return resolveAoECircle(hitBuf, hitBuf[0].Target.TargetPos(),
		caster.CommitterID(), splashTargets, ctx.Obstacles,
		def.SplashRadius, splashDmg, ctx.SourceType)
}

// applyBDFlowBonus applies the BD Flow multiplier bonus damage to all hits in place.
func applyBDFlowBonus(hitBuf []DamageResult, p *entity.Player, def *AbilityDef) {
	flow := getFlowState(p)
	flowMult := flow.RecordTransition(def.OriginConfig, def.DestConfig, p.GearStats.Mastery)
	if flowMult <= 1.0 {
		return
	}
	for i := range hitBuf {
		bonus := hitBuf[i].Amount * (flowMult - 1.0)
		if t := hitBuf[i].Target; t != nil && t.TargetAlive() {
			if dealt := t.TargetApplyDamage(bonus); dealt > 0 {
				hitBuf[i].Amount += dealt
			}
		}
	}
}

// applyBDResonanceBonus consumes the resonance charge and amplifies all hit amounts.
func applyBDResonanceBonus(hitBuf []DamageResult, p *entity.Player, factor float32) {
	if res := p.Resources["resonance"]; res != nil {
		res.Current = 0
		for i := range hitBuf {
			bonus := hitBuf[i].Amount * factor
			if t := hitBuf[i].Target; t != nil && t.TargetAlive() {
				if dealt := t.TargetApplyDamage(bonus); dealt > 0 {
					hitBuf[i].Amount += dealt
				}
			}
		}
	}
}

// applyPostCommitEffects applies all side-effects that happen after a successful player commit:
// self-buffs, shield grants, DoTs, debuffs, config transition, cooldown/GCD, and state.
func applyPostCommitEffects(abilityID string, def *AbilityDef, p *entity.Player, events []DamageResult) {
	for _, buff := range def.SelfBuffs {
		p.AddBuff(entity.ActiveBuff{
			ID:       buff.ID,
			Type:     buff.Type,
			Value:    buff.Value,
			Duration: buff.Duration,
		})
	}

	if def.ShieldGrant > 0 || def.ShieldScalesWithDamage {
		applyShieldGrant(def, p, events)
	}

	applyTargetDoTs(def, p, events)
	applyTargetDebuffs(def, p, events)
	applyConfigTransition(def, p)
	applyCommitTimers(abilityID, def, p)
}

// applyTargetDoTs registers per-target DoTs on the player for each hit event.
func applyTargetDoTs(def *AbilityDef, p *entity.Player, events []DamageResult) {
	for _, dot := range def.TargetDoTs {
		for _, evt := range events {
			p.DoTs = append(p.DoTs, entity.ActiveDoT{
				EnemyID:    evt.TargetID,
				SourcePeer: p.ID,
				AbilityID:  def.Name,
				Damage:     dot.Damage,
				Remaining:  dot.Duration,
				Interval:   dot.Interval,
				TickTimer:  dot.Interval,
			})
		}
	}
}

// applyTargetDebuffs applies debuffs to each enemy hit by the ability.
func applyTargetDebuffs(def *AbilityDef, p *entity.Player, events []DamageResult) {
	for _, debuff := range def.TargetDebuffs {
		for _, evt := range events {
			if enemy, ok := evt.Target.(*entity.Enemy); ok && enemy.Alive {
				enemy.AddDebuff(entity.ActiveDebuff{
					ID:       debuff.ID,
					Type:     debuff.Type,
					Value:    debuff.Value,
					Duration: debuff.Duration,
					SourceID: p.ID,
				})
			}
		}
	}
}

// applyConfigTransition updates the player config and grants BD resonance charge on transition.
func applyConfigTransition(def *AbilityDef, p *entity.Player) {
	if def.DestConfig < 0 {
		return
	}
	p.Config = def.DestConfig
	if p.ClassID != entity.ClassBladeDancer {
		return
	}
	if res := p.Resources["resonance"]; res != nil {
		gain := float32(10.0) * (1.0 + p.GearStats.Identity/100.0)
		res.Current += gain
		if res.Current > res.Max {
			res.Current = res.Max
		}
	}
}

// applyCommitTimers sets the cooldown, GCD, and attack state following a successful commit.
func applyCommitTimers(abilityID string, def *AbilityDef, p *entity.Player) {
	if def.Cooldown > 0 {
		cd := def.Cooldown
		for i := range p.Buffs {
			if p.Buffs[i].Type == entity.BuffCooldownMult {
				cd *= p.Buffs[i].Value
			}
		}
		p.Cooldowns[abilityID] = cd
	}

	if def.GCD > 0 {
		p.GCDTimer = def.GCD
	}
	if def.LockoutDuration > 0 {
		p.GCDTimer = def.LockoutDuration
	}

	if def.Hit.Type != HitNone || def.LockoutDuration > 0 {
		p.State = entity.PlayerStateAttack
	}
}

// applyShieldGrant updates the player's shield resource after a commit.
func applyShieldGrant(def *AbilityDef, p *entity.Player, events []DamageResult) {
	shield, ok := p.Resources["shield"]
	if !ok {
		return
	}
	grant := def.ShieldGrant
	if def.ShieldScalesWithDamage {
		var totalDmg float32
		for _, ev := range events {
			totalDmg += ev.Amount
		}
		grant = totalDmg * def.ShieldPerDamage
	}
	shield.Current += grant
	if def.ShieldCap > 0 && shield.Current > def.ShieldCap {
		shield.Current = def.ShieldCap
	} else if shield.Current > shield.Max {
		shield.Current = shield.Max
	}
}

// TickPlayer updates cooldowns, buffs, resources, and DoTs for one player.
// Returns damage events from DoT ticks.
func (eng *Engine) TickPlayer(p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	if !p.Alive {
		return nil
	}

	eng.tickBuf = eng.tickBuf[:0]

	// Tick invincibility timer (dodge i-frames)
	if p.Invincible && p.InvincibleTimer > 0 {
		p.InvincibleTimer -= dt
		if p.InvincibleTimer <= 0 {
			p.Invincible = false
			p.InvincibleTimer = 0
		}
	}

	// Tempo: cooldowns drain faster, GCD resolves faster, regen is faster
	tempoMult := p.TempoMult()

	tickCooldownsAndGCD(p, dt, tempoMult)
	eng.maybeResetAttackState(p)

	// Tick custom handlers
	for name, fn := range eng.tickHandlers {
		results := fn(eng, p, dt, ctx)
		for i := range results {
			if results[i].AbilityID == "" {
				results[i].AbilityID = name
			}
		}
		eng.tickBuf = append(eng.tickBuf, results...)
	}

	eng.tickBuffs(p, dt)

	// Tick Confluence (Arcanotechnicien class mechanic)
	if p.Confluence != nil {
		p.Confluence.Tick(dt)
	}

	// Tick VitalCharge expiry (Life Swap stored drain)
	if p.VitalChargeTimer > 0 {
		p.VitalChargeTimer -= dt
		if p.VitalChargeTimer <= 0 {
			p.VitalCharge = 0
			p.VitalChargeTimer = 0
		}
	}

	tickResources(p, dt, tempoMult)

	eng.tickDoTs(p, dt, ctx)

	return eng.tickBuf
}

// tickCooldownsAndGCD drains all ability cooldowns and the global cooldown timer.
func tickCooldownsAndGCD(p *entity.Player, dt float32, tempoMult float32) {
	for id, cd := range p.Cooldowns {
		cd -= dt * tempoMult
		if cd <= 0 {
			delete(p.Cooldowns, id)
		} else {
			p.Cooldowns[id] = cd
		}
	}
	if p.GCDTimer > 0 {
		p.GCDTimer -= dt * tempoMult
		if p.GCDTimer < 0 {
			p.GCDTimer = 0
		}
	}
}

// maybeResetAttackState transitions the player from attack state back to move
// once the GCD has expired and no attack cooldowns remain.
func (eng *Engine) maybeResetAttackState(p *entity.Player) {
	if p.State != entity.PlayerStateAttack || p.GCDTimer > 0 {
		return
	}
	for id, cd := range p.Cooldowns {
		if cd <= 0 {
			continue
		}
		def := eng.abilities[id]
		if def != nil && (def.LockoutDuration > 0 || def.Hit.Type != HitNone) {
			return
		}
	}
	p.State = entity.PlayerStateMove
}

// tickBuffs advances buff durations and removes expired ones, logging each expiry.
func (eng *Engine) tickBuffs(p *entity.Player, dt float32) {
	alive := p.Buffs[:0]
	for i := range p.Buffs {
		if p.Buffs[i].Duration > 0 {
			p.Buffs[i].Duration -= dt
			if p.Buffs[i].Duration <= 0 {
				if eng.logDebug {
					eng.logger.Debug("ability.buff.expired",
						"peer", p.ID,
						"buff", p.Buffs[i].ID,
						"type", p.Buffs[i].Type,
					)
				}
				continue
			}
		}
		alive = append(alive, p.Buffs[i])
	}
	p.Buffs = alive
}

// tickResources regenerates all player resources, handling delay timers and
// per-school flux commitment pools.
func tickResources(p *entity.Player, dt float32, tempoMult float32) {
	hasFluxCommit := p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0
	for name, r := range p.Resources {
		// Skip generic flux regen for FluxCommitment players — handled by pool-based regen below.
		if name == entity.ResourceFlux && hasFluxCommit {
			continue
		}
		if r.DelayTimer > 0 {
			r.DelayTimer -= dt
			if r.DelayTimer < 0 {
				if r.Regen > 0 {
					r.Current += r.Regen * (-r.DelayTimer) * tempoMult
					if r.Current > r.Max {
						r.Current = r.Max
					}
				}
				r.DelayTimer = 0
			}
			continue
		}
		if r.Regen != 0 {
			r.Current += r.Regen * dt * tempoMult
			if r.Current > r.Max {
				r.Current = r.Max
			}
			if r.Current < 0 {
				r.Current = 0
			}
		}
	}

	// Per-school flux regen for FluxCommitment players.
	if hasFluxCommit {
		p.FluxCommit.TickRegen(dt * tempoMult)
		p.SyncFluxAggregate()
	}
}

// tickDoTs advances all active DoTs, fires damage on tick intervals, and removes expired DoTs.
func (eng *Engine) tickDoTs(p *entity.Player, dt float32, ctx *TickContext) {
	aliveDoTs := p.DoTs[:0]
	for i := range p.DoTs {
		dot := &p.DoTs[i]
		dot.Remaining -= dt
		if dot.Remaining <= 0 {
			continue
		}
		dot.TickTimer -= dt
		if dot.TickTimer <= 0 {
			dot.TickTimer += dot.Interval
			eng.fireDotTick(p, dot, ctx)
		}
		aliveDoTs = append(aliveDoTs, *dot)
	}
	p.DoTs = aliveDoTs
}

// fireDotTick applies one DoT damage tick to the matching target.
func (eng *Engine) fireDotTick(p *entity.Player, dot *entity.ActiveDoT, ctx *TickContext) {
	if ctx == nil {
		return
	}
	for _, t := range ctx.Targets {
		if t == nil || t.TargetID() != dot.EnemyID || !t.TargetAlive() {
			continue
		}
		dealt := t.TargetApplyDamage(dot.Damage)
		if dealt > 0 {
			eng.tickBuf = append(eng.tickBuf, DamageResult{
				TargetID:   t.TargetID(),
				SourceID:   dot.SourcePeer,
				Amount:     dealt,
				HitPos:     t.TargetPos().Add(entity.Vec3{Y: 1.0}),
				SourceType: combat.SourcePlayerAttack,
				AbilityID:  dot.AbilityID,
				Target:     t,
			})
			if th, ok := t.(entity.Threateable); ok {
				th.AddThreat(dot.SourcePeer, dealt)
			}
			if eng.logDebug {
				eng.logger.Debug("ability.dot.tick",
					"peer", p.ID,
					"target", t.TargetID(),
					"damage", dealt,
					"remaining", dot.Remaining,
				)
			}
		}
		break
	}
}
