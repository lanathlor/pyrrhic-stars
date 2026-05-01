package ability

import (
	"io"
	"log/slog"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// HandlerFunc is a Go function that handles complex ability execution.
type HandlerFunc func(eng *Engine, ctx *CastContext) CastResult

// TickHandlerFunc is a Go function that handles per-tick logic for an ability.
type TickHandlerFunc func(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult

// CastContext carries the state needed to resolve an ability.
type CastContext struct {
	Player    *entity.Player
	Enemies   []*entity.Enemy
	Obstacles []combat.Obstacle
}

// TickContext carries the state needed for per-tick ability updates.
type TickContext struct {
	Enemies   []*entity.Enemy
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

	// hitBuf is a reusable scratch buffer for hit resolution results.
	// Valid only until the next Cast or TickPlayer call.
	hitBuf []DamageResult
}

// NewEngine creates an engine and registers all abilities and handlers.
// Pass nil for logger to discard all log output.
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	eng := &Engine{
		abilities:    make(map[string]*AbilityDef),
		handlers:     make(map[string]HandlerFunc),
		tickHandlers: make(map[string]TickHandlerFunc),
		logger:       logger,
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

// Cast validates and executes an ability for a player.
// The returned CastResult.Events slice is backed by an internal buffer and is
// only valid until the next Cast or TickPlayer call.
func (eng *Engine) Cast(abilityID string, ctx *CastContext) CastResult {
	eng.hitBuf = eng.hitBuf[:0]

	def := eng.abilities[abilityID]
	if def == nil {
		eng.logger.Warn("ability.cast.rejected",
			"ability", abilityID,
			"reason", "unknown ability",
		)
		return CastResult{Reason: "unknown ability"}
	}

	p := ctx.Player
	log := eng.logger.With("ability", abilityID, "peer", p.PeerID)

	if !p.Alive {
		log.Debug("ability.cast.rejected", "reason", "dead")
		return CastResult{Reason: "dead"}
	}

	// Check GCD
	if p.GCDTimer > 0 {
		log.Debug("ability.cast.rejected", "reason", "gcd", "gcd_remaining", p.GCDTimer)
		return CastResult{Reason: "gcd"}
	}

	// Check per-ability cooldown
	if cd, ok := p.Cooldowns[abilityID]; ok && cd > 0 {
		log.Debug("ability.cast.rejected", "reason", "cooldown", "cd_remaining", cd)
		return CastResult{Reason: "cooldown"}
	}

	// Check BD spell config requirement
	if def.OriginConfig >= 0 && def.OriginConfig != p.Config {
		log.Debug("ability.cast.rejected", "reason", "wrong config",
			"need", def.OriginConfig, "have", p.Config)
		return CastResult{Reason: "wrong config"}
	}

	// Check resource costs (don't spend yet)
	for _, cost := range def.Costs {
		r, ok := p.Resources[cost.Resource]
		if !ok {
			log.Debug("ability.cast.rejected", "reason", "insufficient "+cost.Resource,
				"need", cost.Amount, "have", 0)
			return CastResult{Reason: "insufficient " + cost.Resource}
		}
		if r.Current < cost.Amount {
			log.Debug("ability.cast.rejected", "reason", "insufficient "+cost.Resource,
				"need", cost.Amount, "have", r.Current)
			return CastResult{Reason: "insufficient " + cost.Resource}
		}
	}

	// If this ability has a custom handler, delegate entirely
	if def.Handler != "" {
		fn, ok := eng.handlers[def.Handler]
		if !ok {
			log.Warn("ability.cast.rejected", "reason", "handler not found", "handler", def.Handler)
			return CastResult{Reason: "handler not found: " + def.Handler}
		}
		result := fn(eng, ctx)
		if result.OK {
			var totalDmg float32
			for _, ev := range result.Events {
				totalDmg += ev.Amount
			}
			log.Info("ability.cast", "handler", def.Handler, "hits", len(result.Events), "damage", totalDmg)
		} else {
			log.Debug("ability.cast.rejected", "handler", def.Handler, "reason", result.Reason)
		}
		return result
	}

	// Spend resources
	for _, cost := range def.Costs {
		p.SpendResource(cost.Resource, cost.Amount)
	}

	// Resolve hit
	eng.hitBuf = resolveHit(eng.hitBuf, def, p, ctx.Enemies, ctx.Obstacles)
	events := eng.hitBuf

	// Apply self buffs
	for _, buff := range def.SelfBuffs {
		p.AddBuff(entity.ActiveBuff{
			ID:       buff.ID,
			Type:     buff.Type,
			Value:    buff.Value,
			Duration: buff.Duration,
		})
	}

	// Grant shield
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

	// Apply DoTs to hit targets
	for _, dot := range def.TargetDoTs {
		for _, evt := range events {
			p.DoTs = append(p.DoTs, entity.ActiveDoT{
				EnemyID:    evt.TargetID,
				SourcePeer: p.PeerID,
				Damage:     dot.Damage,
				Remaining:  dot.Duration,
				Interval:   dot.Interval,
				TickTimer:  dot.Interval,
			})
		}
	}

	// BD config transition
	if def.DestConfig >= 0 {
		p.Config = def.DestConfig
	}

	// Set cooldown
	if def.Cooldown > 0 {
		cd := def.Cooldown
		for i := range p.Buffs {
			if p.Buffs[i].Type == entity.BuffCooldownMult {
				cd *= p.Buffs[i].Value
			}
		}
		p.Cooldowns[abilityID] = cd
	}

	// Set GCD
	if def.GCD > 0 {
		p.GCDTimer = def.GCD
	}

	// Set lockout
	if def.LockoutDuration > 0 {
		p.GCDTimer = def.LockoutDuration
	}

	// Set attack state
	if def.Hit.Type != HitNone || def.LockoutDuration > 0 {
		p.State = entity.PlayerStateAttack
	}

	// Log successful cast
	var totalDmg float32
	for _, ev := range events {
		totalDmg += ev.Amount
	}
	attrs := []any{"hits", len(events), "damage", totalDmg}
	if def.DestConfig >= 0 {
		attrs = append(attrs, "config", p.Config)
	}
	if len(def.SelfBuffs) > 0 {
		attrs = append(attrs, "buffs_applied", len(def.SelfBuffs))
	}
	if def.ShieldGrant > 0 {
		attrs = append(attrs, "shield", p.GetResource("shield"))
	}
	log.Info("ability.cast", attrs...)

	return CastResult{OK: true, Events: events}
}

// TickPlayer updates cooldowns, buffs, resources, and DoTs for one player.
// Returns damage events from DoT ticks.
func (eng *Engine) TickPlayer(p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	if !p.Alive {
		return nil
	}

	var events []DamageResult

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
			events = append(events, results...)
		}
	}

	// Tick buffs
	log := eng.logger.With("peer", p.PeerID)
	alive := p.Buffs[:0]
	for i := range p.Buffs {
		if p.Buffs[i].Duration > 0 {
			p.Buffs[i].Duration -= dt
			if p.Buffs[i].Duration <= 0 {
				log.Debug("ability.buff.expired",
					"buff", p.Buffs[i].ID,
					"type", p.Buffs[i].Type,
				)
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
				for _, e := range ctx.Enemies {
					if e != nil && e.ID == dot.EnemyID && e.Alive {
						dealt, _ := e.ApplyDamage(dot.Damage)
						if dealt > 0 {
							events = append(events, DamageResult{
								TargetID:   e.ID,
								SourceID:   dot.SourcePeer,
								Amount:     dealt,
								HitPos:     e.Position.Add(entity.Vec3{Y: 1.0}),
								SourceType: combat.SourcePlayerAttack,
								Enemy:      e,
							})
							e.AddThreat(dot.SourcePeer, dealt)
							log.Debug("ability.dot.tick",
								"target", e.ID,
								"damage", dealt,
								"remaining", dot.Remaining,
							)
						}
						break
					}
				}
			}
		}
		aliveDoTs = append(aliveDoTs, *dot)
	}
	p.DoTs = aliveDoTs

	return events
}
