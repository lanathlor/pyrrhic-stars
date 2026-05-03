package ability

import (
	"context"
	"log/slog"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// nopHandler is a slog.Handler that discards all output with zero allocations.
type nopHandler struct{}

func (nopHandler) Enabled(context.Context, slog.Level) bool { return false }
func (nopHandler) Handle(context.Context, slog.Record) error { return nil }
func (nopHandler) WithAttrs([]slog.Attr) slog.Handler        { return nopHandler{} }
func (nopHandler) WithGroup(string) slog.Handler              { return nopHandler{} }

// HandlerFunc is a Go function that handles complex ability execution.
type HandlerFunc func(eng *Engine, ctx *CastContext) CastResult

// TickHandlerFunc is a Go function that handles per-tick logic for an ability.
type TickHandlerFunc func(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult

// CastContext carries the state needed to resolve an ability.
type CastContext struct {
	Caster     entity.Caster
	Targets    []entity.Target
	Obstacles  []combat.Obstacle
	SourceType uint8 // combat.SourcePlayerAttack, SourceEnemyMelee, etc.
}

// TickContext carries the state needed for per-tick ability updates.
type TickContext struct {
	Targets   []entity.Target
	Obstacles []combat.Obstacle
}

// Common cast failure reasons.
const ReasonInsufficientStamina = "insufficient stamina"

// CastResult is returned by the engine after attempting to cast an ability.
type CastResult struct {
	OK     bool
	Events []DamageResult
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
	// Valid only until the next Cast or TickPlayer call.
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

// Cast looks up an ability by ID and executes it.
// The returned CastResult.Events slice is backed by an internal buffer and is
// only valid until the next Cast or TickPlayer call.
func (eng *Engine) Cast(abilityID string, ctx *CastContext) CastResult {
	def := eng.abilities[abilityID]
	if def == nil {
		eng.logger.Warn("ability.cast.rejected",
			"ability", abilityID,
			"reason", "unknown ability",
		)
		return CastResult{Reason: "unknown ability"}
	}
	return eng.doCast(abilityID, def, ctx)
}

// CastDef executes an ability from a provided definition (not looked up by ID).
// Use this when the caller has a modified/resolved copy (e.g. enemy phase overrides).
func (eng *Engine) CastDef(def *AbilityDef, ctx *CastContext) CastResult {
	return eng.doCast(def.ID, def, ctx)
}

func (eng *Engine) doCast(abilityID string, def *AbilityDef, ctx *CastContext) CastResult {
	eng.hitBuf = eng.hitBuf[:0]

	caster := ctx.Caster

	if !caster.CasterAlive() {
		if eng.logDebug {
			eng.logger.Debug("ability.cast.rejected",
				"ability", abilityID, "id", caster.CasterID(), "reason", "dead")
		}
		return CastResult{Reason: "dead"}
	}

	// Player-specific validation
	p, isPlayer := caster.(*entity.Player)
	if isPlayer {
		if p.GCDTimer > 0 {
			if eng.logDebug {
				eng.logger.Debug("ability.cast.rejected",
					"ability", abilityID, "id", caster.CasterID(),
					"reason", "gcd", "gcd_remaining", p.GCDTimer)
			}
			return CastResult{Reason: "gcd"}
		}

		if cd, ok := p.Cooldowns[abilityID]; ok && cd > 0 {
			if eng.logDebug {
				eng.logger.Debug("ability.cast.rejected",
					"ability", abilityID, "id", caster.CasterID(),
					"reason", "cooldown", "cd_remaining", cd)
			}
			return CastResult{Reason: "cooldown"}
		}

		if def.OriginConfig >= 0 && def.OriginConfig != p.Config {
			if eng.logDebug {
				eng.logger.Debug("ability.cast.rejected",
					"ability", abilityID, "id", caster.CasterID(),
					"reason", "wrong config", "need", def.OriginConfig, "have", p.Config)
			}
			return CastResult{Reason: "wrong config"}
		}

		for _, cost := range def.Costs {
			r, ok := p.Resources[cost.Resource]
			if !ok {
				if eng.logDebug {
					eng.logger.Debug("ability.cast.rejected",
						"ability", abilityID, "id", caster.CasterID(),
						"reason", "insufficient "+cost.Resource, "need", cost.Amount, "have", 0)
				}
				return CastResult{Reason: "insufficient " + cost.Resource}
			}
			if r.Current < cost.Amount {
				if eng.logDebug {
					eng.logger.Debug("ability.cast.rejected",
						"ability", abilityID, "id", caster.CasterID(),
						"reason", "insufficient "+cost.Resource, "need", cost.Amount, "have", r.Current)
				}
				return CastResult{Reason: "insufficient " + cost.Resource}
			}
		}
	}

	// If this ability has a custom handler, delegate entirely
	if def.Handler != "" {
		fn, ok := eng.handlers[def.Handler]
		if !ok {
			eng.logger.Warn("ability.cast.rejected",
				"ability", abilityID, "id", caster.CasterID(),
				"reason", "handler not found", "handler", def.Handler)
			return CastResult{Reason: "handler not found: " + def.Handler}
		}
		result := fn(eng, ctx)
		if eng.logInfo {
			if result.OK {
				var totalDmg float32
				for _, ev := range result.Events {
					totalDmg += ev.Amount
				}
				eng.logger.Info("ability.cast",
					"ability", abilityID, "id", caster.CasterID(),
					"handler", def.Handler, "hits", len(result.Events), "damage", totalDmg)
			} else if eng.logDebug {
				eng.logger.Debug("ability.cast.rejected",
					"ability", abilityID, "id", caster.CasterID(),
					"handler", def.Handler, "reason", result.Reason)
			}
		}
		return result
	}

	// Spend resources (player only)
	if isPlayer {
		for _, cost := range def.Costs {
			p.SpendResource(cost.Resource, cost.Amount)
		}
	}

	// Resolve hit
	eng.hitBuf = resolveHit(eng.hitBuf, def, caster, ctx.Targets, ctx.Obstacles, ctx.SourceType)
	events := eng.hitBuf

	// Player-specific post-cast effects
	if isPlayer {
		for _, buff := range def.SelfBuffs {
			p.AddBuff(entity.ActiveBuff{
				ID:       buff.ID,
				Type:     buff.Type,
				Value:    buff.Value,
				Duration: buff.Duration,
			})
		}

		if def.ShieldGrant > 0 {
			if shield, ok := p.Resources["shield"]; ok {
				shield.Current += def.ShieldGrant
				if def.ShieldCap > 0 && shield.Current > def.ShieldCap {
					shield.Current = def.ShieldCap
				} else if shield.Current > shield.Max {
					shield.Current = shield.Max
				}
			}
		}

		for _, dot := range def.TargetDoTs {
			for _, evt := range events {
				p.DoTs = append(p.DoTs, entity.ActiveDoT{
					EnemyID:    evt.TargetID,
					SourcePeer: p.ID,
					Damage:     dot.Damage,
					Remaining:  dot.Duration,
					Interval:   dot.Interval,
					TickTimer:  dot.Interval,
				})
			}
		}

		if def.DestConfig >= 0 {
			p.Config = def.DestConfig
		}

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

	if eng.logInfo {
		var totalDmg float32
		for _, ev := range events {
			totalDmg += ev.Amount
		}
		eng.logger.Info("ability.cast",
			"ability", abilityID, "id", caster.CasterID(),
			"hits", len(events), "damage", totalDmg)
	}

	return CastResult{OK: true, Events: events}
}

// TickPlayer updates cooldowns, buffs, resources, and DoTs for one player.
// Returns damage events from DoT ticks.
func (eng *Engine) TickPlayer(p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	if !p.Alive {
		return nil
	}

	eng.tickBuf = eng.tickBuf[:0]

	// Tick cooldowns
	for id, cd := range p.Cooldowns {
		cd -= dt
		if cd <= 0 {
			delete(p.Cooldowns, id)
		} else {
			p.Cooldowns[id] = cd
		}
	}

	// Tick GCD
	if p.GCDTimer > 0 {
		p.GCDTimer -= dt
		if p.GCDTimer < 0 {
			p.GCDTimer = 0
		}
	}

	// Reset attack state when GCD expires
	if p.State == entity.PlayerStateAttack && p.GCDTimer <= 0 {
		hasAttackCooldown := false
		for id, cd := range p.Cooldowns {
			if cd <= 0 {
				continue
			}
			def := eng.abilities[id]
			if def != nil && (def.LockoutDuration > 0 || def.Hit.Type != HitNone) {
				hasAttackCooldown = true
				break
			}
		}
		if !hasAttackCooldown {
			p.State = entity.PlayerStateMove
		}
	}

	// Tick custom handlers
	for name, fn := range eng.tickHandlers {
		if _, ok := p.AbilityState[name]; ok {
			results := fn(eng, p, dt, ctx)
			eng.tickBuf = append(eng.tickBuf, results...)
		}
	}

	// Tick buffs
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

	// Tick resources
	for _, r := range p.Resources {
		if r.DelayTimer > 0 {
			r.DelayTimer -= dt
			if r.DelayTimer < 0 {
				if r.Regen > 0 {
					r.Current += r.Regen * (-r.DelayTimer)
					if r.Current > r.Max {
						r.Current = r.Max
					}
				}
				r.DelayTimer = 0
			}
			continue
		}
		if r.Regen != 0 {
			r.Current += r.Regen * dt
			if r.Current > r.Max {
				r.Current = r.Max
			}
			if r.Current < 0 {
				r.Current = 0
			}
		}
	}

	// Tick DoTs
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
			if ctx != nil {
				for _, t := range ctx.Targets {
					if t != nil && t.TargetID() == dot.EnemyID && t.TargetAlive() {
						dealt := t.TargetApplyDamage(dot.Damage)
						if dealt > 0 {
							eng.tickBuf = append(eng.tickBuf, DamageResult{
								TargetID:   t.TargetID(),
								SourceID:   dot.SourcePeer,
								Amount:     dealt,
								HitPos:     t.TargetPos().Add(entity.Vec3{Y: 1.0}),
								SourceType: combat.SourcePlayerAttack,
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
			}
		}
		aliveDoTs = append(aliveDoTs, *dot)
	}
	p.DoTs = aliveDoTs

	return eng.tickBuf
}
