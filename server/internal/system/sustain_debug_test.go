package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
)

func TestVitalDrainSustainDamage(t *testing.T) {
	eng := ability.NewEngine(nil)
	lvl := testArenaLevel(t)

	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.SpawnTick = 0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
	p.RotationY = 0

	enemy := entity.NewEnemy(1, 1000, "test_boss")
	enemy.Alive = true
	enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	enemy.State = entity.EnemyChase

	w := World{
		Players:        map[uint16]*entity.Player{1: p},
		Enemies:        []*entity.Enemy{enemy},
		Level:          lvl,
		AbilityEngine:  eng,
		PatternEngine:  combat.NewPatternEngine(),
		AbilityRunners: make(map[uint16]*ability.PlayerAbilityRunner),
		CombatLogSink:  combatlog.NullSink{},
		SendBuf:        make([]byte, 0, 256),
		DamageBuf:      make([]byte, 0, 256),
		GameFlowBuf:    make([]byte, 0, 256),
	}

	def := eng.GetAbility("vital_drain")
	if def == nil {
		t.Fatal("vital_drain not registered")
	}

	// Start runner in commit
	runner := &ability.PlayerAbilityRunner{}
	w.AbilityRunners[1] = runner
	runner.Start(def)

	cs := &CombatSystem{}

	// Tick through commit (1.0s) + execute (0.1s) = 22 ticks at 0.05
	for i := range 25 {
		w.TickNum = uint32(i + 1)
		w.DamageEvents = w.DamageEvents[:0]
		cs.Tick(&w, 0.05)
	}

	t.Logf("after 1.25s: runner.Phase=%d (Sustain=%d)", runner.Phase, ability.PRunnerSustain)
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("expected sustain phase, got %d", runner.Phase)
	}

	// Now tick sustain for 1 second — should get 2 sustain ticks
	totalDmg := float32(0)
	for i := range 20 {
		w.TickNum = uint32(26 + i)
		w.DamageEvents = w.DamageEvents[:0]
		cs.Tick(&w, 0.05)
		for _, ev := range w.DamageEvents {
			if ev.SourceType == combat.SourcePlayerAttack {
				totalDmg += ev.Amount
				t.Logf("  sustain damage: %.1f to enemy %d", ev.Amount, ev.TargetPeerID)
			}
		}
	}

	t.Logf("total sustain damage in 1s: %.1f", totalDmg)
	t.Logf("enemy HP: %.1f / %.1f", enemy.Health, enemy.MaxHealth)
	if totalDmg == 0 {
		t.Fatal("expected sustain damage > 0")
	}
}
