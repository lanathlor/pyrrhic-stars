package enemyai

import (
	"log/slog"
	"math/rand/v2"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
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

// ForceCast unconditionally interrupts any current ability and starts the given one.
func (b *Brain) ForceCast(abilityID string) bool {
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

// Tick advances the BT by dt seconds. Returns damage events to emit.
func (b *Brain) Tick(dt float32, players []*entity.Player,
	obstacles []combat.Obstacle,
	spawnProjectile func(pos, dir entity.Vec3, speed, damage, lifetime float32),
	castPattern func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3),
) []combat.DamageEvent {
	e := b.enemy

	e.StateTimer -= dt
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
