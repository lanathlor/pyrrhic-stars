package bosstest

import (
	"fmt"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

// FakePlayer defines a player for scenario testing.
type FakePlayer struct {
	ID    uint16
	Pos   entity.Vec3
	Class string
}

// Scenario configures a single-moment boss behavior test.
type Scenario struct {
	bossName string
	phase    int
	hpPct    float32
	players  []FakePlayer
	seed     uint64
	ticks    int
}

// NewScenario creates a scenario builder for the named boss.
func NewScenario(boss string) *Scenario {
	return &Scenario{
		bossName: boss,
		phase:    1,
		hpPct:    1.0,
		ticks:    1,
		seed:     42,
	}
}

// Phase sets the boss phase (1-indexed).
func (s *Scenario) Phase(p int) *Scenario {
	s.phase = p
	return s
}

// HP sets the boss HP percentage (0.0-1.0).
func (s *Scenario) HP(pct float32) *Scenario {
	s.hpPct = pct
	return s
}

// Players sets the player list for this scenario.
func (s *Scenario) Players(players ...FakePlayer) *Scenario {
	s.players = players
	return s
}

// Seed sets the RNG seed for deterministic behavior.
func (s *Scenario) Seed(seed uint64) *Scenario {
	s.seed = seed
	return s
}

// Ticks sets how many ticks to simulate (default 1).
func (s *Scenario) Ticks(n int) *Scenario {
	s.ticks = n
	return s
}

// Run executes the scenario and returns the result.
func (s *Scenario) Run() *ScenarioResult {
	def := enemyai.DefRegistry[s.bossName]
	if def == nil {
		panic(fmt.Sprintf("Scenario.Run: boss %q not in DefRegistry", s.bossName))
	}

	enemy := entity.NewEnemy(0, def.MaxHealth, def.Name)
	enemy.Alive = true
	enemy.IsBoss = true
	enemy.LeashRadius = 100
	enemy.AggroRadius = 50

	// Apply damage to reach desired HP, triggering phase transitions naturally.
	if s.hpPct < 1.0 {
		dmgNeeded := def.MaxHealth * (1.0 - s.hpPct)
		enemy.ApplyDamage(dmgNeeded)
	}
	// If enemy died from the HP setup (hpPct=0), leave it dead.
	// Otherwise ensure it's in a combat-ready state after transition.
	if enemy.Alive && enemy.State == entity.EnemyPhaseTransition {
		enemy.StateTimer = 0 // skip transition animation
	}

	engine := ability.NewEngine(nil)
	brain := enemyai.NewBrainSeeded(def, enemy, engine, s.seed)
	brain.BoundsMinX = -20
	brain.BoundsMaxX = 20
	brain.BoundsMinZ = -20
	brain.BoundsMaxZ = 20

	// Build players
	players := make([]*entity.Player, len(s.players))
	for i, fp := range s.players {
		p := entity.NewPlayer(fp.ID, fp.Class)
		p.Position = fp.Pos
		p.Health = p.MaxHealth
		p.Alive = true
		players[i] = p
	}

	// Instrument the tree for node-level assertions
	// Note: for scenarios we tick the brain directly (tree is internal to brain)
	// So we collect damage events and state changes instead.

	var allEvents []combat.DamageEvent
	var obstacles []combat.Obstacle

	for range s.ticks {
		events := brain.Tick(defaultDt, players, obstacles,
			func(_, _ entity.Vec3, _, _, _ float32) {},
			func(_ *combat.PatternDef, _ string, _, _ entity.Vec3) {},
		)
		allEvents = append(allEvents, events...)
	}

	return &ScenarioResult{
		Enemy:  enemy,
		Events: allEvents,
	}
}

// ScenarioResult holds the output of a scenario run.
type ScenarioResult struct {
	Enemy  *entity.Enemy
	Events []combat.DamageEvent
}

// Summary returns a one-line description of the boss state after the run.
func (r *ScenarioResult) Summary() string {
	if !r.Enemy.Alive {
		return fmt.Sprintf("dead (phase=%d)", r.Enemy.Phase)
	}
	hp := r.Enemy.Health / r.Enemy.MaxHealth * 100
	return fmt.Sprintf("alive hp=%.0f%% phase=%d state=%d dmg_events=%d",
		hp, r.Enemy.Phase, r.Enemy.State, len(r.Events))
}

// AssertState checks the enemy ended in the expected state.
func (r *ScenarioResult) AssertState(t *testing.T, expected entity.EnemyState) {
	t.Helper()
	if r.Enemy.State != expected {
		t.Errorf("enemy state = %d, want %d", r.Enemy.State, expected)
	}
}

// AssertAbilityActive checks the enemy is executing the named ability.
func (r *ScenarioResult) AssertAbilityActive(t *testing.T, abilityIdx int) {
	t.Helper()
	if r.Enemy.ActiveAbility != abilityIdx {
		t.Errorf("active ability = %d, want %d", r.Enemy.ActiveAbility, abilityIdx)
	}
}

// AssertDamageDealt checks that at least one damage event was emitted.
func (r *ScenarioResult) AssertDamageDealt(t *testing.T) {
	t.Helper()
	if len(r.Events) == 0 {
		t.Error("expected damage events, got none")
	}
}

// AssertNoDamage checks that no damage events were emitted.
func (r *ScenarioResult) AssertNoDamage(t *testing.T) {
	t.Helper()
	if len(r.Events) > 0 {
		t.Errorf("expected no damage events, got %d", len(r.Events))
	}
}

// AssertPhase checks the enemy is in the expected phase.
func (r *ScenarioResult) AssertPhase(t *testing.T, expected int) {
	t.Helper()
	if r.Enemy.Phase != expected {
		t.Errorf("enemy phase = %d, want %d", r.Enemy.Phase, expected)
	}
}

// AssertAlive checks the enemy is alive.
func (r *ScenarioResult) AssertAlive(t *testing.T) {
	t.Helper()
	if !r.Enemy.Alive {
		t.Error("expected enemy to be alive")
	}
}

// AssertDead checks the enemy is dead.
func (r *ScenarioResult) AssertDead(t *testing.T) {
	t.Helper()
	if r.Enemy.Alive {
		t.Error("expected enemy to be dead")
	}
}
