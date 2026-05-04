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

	// Apply velocity
	e.Position = e.Position.Add(e.Velocity.Scale(dt))

	return b.events
}
