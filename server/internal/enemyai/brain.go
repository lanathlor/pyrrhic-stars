package enemyai

import (
	"log/slog"
	"math/rand/v2"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/overflux"
)

// Brain drives one enemy instance using a behavior tree.
type Brain struct {
	def    *EnemyDef
	enemy  *entity.Enemy
	engine *ability.Engine
	tree   *bt.Tree
	bb     *Blackboard
	ctx    *EntityContext
	events []combat.DamageEvent
	rng    *rand.Rand

	// Logger enables BT trace logging for this brain. Nil disables logging.
	Logger *slog.Logger

	// Bounds for charge wall-stop detection. Set by zone at init.
	BoundsMinX, BoundsMaxX float32
	BoundsMinZ, BoundsMaxZ float32
}

// NewBrain creates a brain for the given enemy.
func NewBrain(def *EnemyDef, enemy *entity.Enemy, engine *ability.Engine) *Brain {
	return NewBrainSeeded(def, enemy, engine, 0)
}

// NewBrainSeeded creates a brain with a specific random seed (for deterministic testing).
func NewBrainSeeded(def *EnemyDef, enemy *entity.Enemy, engine *ability.Engine, seed uint64) *Brain {
	rng := rand.New(rand.NewPCG(seed, seed))
	bb := NewBlackboard()
	ctx := &EntityContext{
		Enemy:  enemy,
		Def:    def,
		Engine: engine,
		BB:     bb,
		Rng:    rng,
		Runner: &AbilityRunner{},
	}
	tree := buildTree(def, ctx)
	// Strip debug name wrappers for production: saves one interface dispatch
	// per node per tick. SetTree re-adds them if instrumentation is needed.
	tree.Root = bt.StripNames(tree.Root)
	return &Brain{
		def:    def,
		enemy:  enemy,
		engine: engine,
		tree:   tree,
		bb:     bb,
		ctx:    ctx,
		rng:    rng,
	}
}

// Enemy returns the brain's enemy.
func (b *Brain) Enemy() *entity.Enemy { return b.enemy }

// ForceCommit unconditionally interrupts any current ability and starts the given one.
func (b *Brain) ForceCommit(abilityID string) bool {
	return b.ctx.Runner.ForceStart(b.ctx, abilityID)
}

// AbilityIDs returns the IDs of all abilities in the enemy definition.
func (b *Brain) AbilityIDs() []string {
	ids := make([]string, len(b.def.Abilities))
	for i := range b.def.Abilities {
		ids[i] = b.def.Abilities[i].ID
	}
	return ids
}

// DefName returns the enemy definition name.
func (b *Brain) DefName() string { return b.def.Name }

// Tree returns the brain's behavior tree root node.
func (b *Brain) Tree() bt.Node { return b.tree.Root }

// SetTree replaces the brain's tree root (used for instrumentation in tests).
func (b *Brain) SetTree(root bt.Node) { b.tree = bt.NewTree(root) }

// ApplyOverfluxVariants resolves active overflux conditions against the enemy's
// variant definitions and applies BT replacements, ability appends, and ability
// overrides. Creates a copy of EnemyDef so the global registry is not mutated.
func (b *Brain) ApplyOverfluxVariants(oflx *overflux.State) {
	if oflx == nil || len(b.def.OverfluxVariants) == 0 {
		return
	}

	slog.Info("applying overflux variants",
		"enemy", b.def.Name,
		"conditions", len(oflx.Conditions),
		"available_variants", len(b.def.OverfluxVariants))

	// Shallow copy the def so we don't mutate the registry entry.
	defCopy := *b.def
	abils := make([]ability.AbilityDef, len(defCopy.Abilities))
	copy(abils, defCopy.Abilities)
	defCopy.Abilities = abils

	var newTree any
	for _, cond := range oflx.Conditions {
		v, ok := defCopy.OverfluxVariants[string(cond.ID)]
		if !ok {
			slog.Debug("no variant for condition", "enemy", b.def.Name, "condition", cond.ID)
			continue
		}

		slog.Info("applying overflux variant",
			"enemy", b.def.Name,
			"condition", cond.ID,
			"tree_override", v.TreeData != nil,
			"new_abilities", len(v.Abilities),
			"ability_overrides", len(v.AbilityOverrides))

		// BT replacement (last one wins if multiple conditions override tree).
		if v.TreeData != nil {
			newTree = v.TreeData
		}

		// Append new abilities.
		defCopy.Abilities = append(defCopy.Abilities, v.Abilities...)

		// Apply per-ability overrides (pattern, cooldown, damage, etc.).
		for abilID, ovr := range v.AbilityOverrides {
			for i := range defCopy.Abilities {
				if defCopy.Abilities[i].ID == abilID {
					applyVariantOverride(&defCopy.Abilities[i], &ovr)
					break
				}
			}
		}
	}

	b.def = &defCopy
	b.ctx.Def = &defCopy

	// Rebuild BT if a variant provided a replacement tree.
	if newTree != nil {
		defCopy.TreeData = newTree
		tree := buildTree(&defCopy, b.ctx)
		tree.Root = bt.StripNames(tree.Root)
		b.tree = tree
	}
}

// applyVariantOverride applies an AbilityOverride directly to an ability def.
// Unlike ResolveAbility (which returns a copy), this mutates in place because
// ApplyOverfluxVariants already works on a copied ability slice.
func applyVariantOverride(ad *ability.AbilityDef, ovr *AbilityOverride) {
	if ovr.CommitTime != nil {
		ad.CommitTime = *ovr.CommitTime
	}
	if ovr.CooldownTime != nil {
		ad.Cooldown = *ovr.CooldownTime
	}
	if ovr.AoERadius != nil {
		ad.Hit.Radius = *ovr.AoERadius
	}
	if ovr.Pattern != nil {
		ad.Pattern = ovr.Pattern
	}
	if ovr.ProjectileCount != nil && ad.Projectile != nil {
		cp := *ad.Projectile
		cp.Count = *ovr.ProjectileCount
		ad.Projectile = &cp
	}
	if ad.Charge != nil && (ovr.ChargeSpeed != nil || ovr.ChargeMaxDistance != nil) {
		cp := *ad.Charge
		if ovr.ChargeSpeed != nil {
			cp.Speed = *ovr.ChargeSpeed
		}
		if ovr.ChargeMaxDistance != nil {
			cp.MaxDistance = *ovr.ChargeMaxDistance
		}
		ad.Charge = &cp
	}
	if ovr.Damage != nil {
		switch {
		case ad.Charge != nil:
			if ovr.ChargeSpeed == nil && ovr.ChargeMaxDistance == nil {
				cp := *ad.Charge
				ad.Charge = &cp
			}
			ad.Charge.Damage = *ovr.Damage
		case ad.Projectile != nil:
			if ovr.ProjectileCount == nil {
				cp := *ad.Projectile
				ad.Projectile = &cp
			}
			ad.Projectile.Damage = *ovr.Damage
		default:
			ad.BaseDamage = *ovr.Damage
		}
	}
}

// Tick advances the BT by dt seconds. Returns damage events to emit.
func (b *Brain) Tick(dt float32, players []*entity.Player,
	obstacles []combat.Obstacle,
	spawnProjectile func(pos, dir entity.Vec3, speed, damage, lifetime float32),
	castPattern func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3),
) []combat.DamageEvent {
	e := b.enemy

	e.StateTimer -= dt
	e.TickDebuffs(dt)
	b.events = b.events[:0]
	b.bb.TickTimers(dt)
	b.ctx.Logger = b.Logger
	b.ctx.Reset(dt, players, obstacles, spawnProjectile, castPattern, &b.events)
	b.ctx.BoundsMinX = b.BoundsMinX
	b.ctx.BoundsMaxX = b.BoundsMaxX
	b.ctx.BoundsMinZ = b.BoundsMinZ
	b.ctx.BoundsMaxZ = b.BoundsMaxZ

	if b.Logger != nil {
		b.Logger.Debug("bt.tick", "enemy", e.ID, "state", e.State, "hp", e.Health, "pos", e.Position)
	}

	b.tree.Tick(b.ctx)
	b.ctx.Runner.Tick(b.ctx)

	// Apply velocity
	e.Position = e.Position.Add(e.Velocity.Scale(dt))

	return b.events
}
