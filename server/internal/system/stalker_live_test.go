package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

// TestLiveSpawnedStalkerFightsBack reproduces the live setup of the Aceras
// General fight: real arena geometry, the room gate sealed, a solo player in
// melee range. The General summons a stalker after its 8s opener; the stalker
// must then close on the player and land talon_rake hits. Live run 2026-07-14
// observed stalkers standing frozen from spawn (no movement, no attacks).
func TestLiveSpawnedStalkerFightsBack(t *testing.T) {
	def := enemyai.DefRegistry["aceras_general"]
	if def == nil {
		t.Fatal("aceras_general not loaded")
	}

	boss := entity.NewEnemy(1016, def.MaxHealth, "aceras_general")
	boss.Alive = true
	boss.IsBoss = true
	boss.BossNum = 2
	boss.BossGateID = testAcerasGateID
	boss.AggroMaxZ = -40
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{Y: -3.9, Z: -58}
	boss.LeashOrigin = boss.Position
	boss.LeashRadius = 50

	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{Y: -3.9, Z: -54}
	p.MaxHealth = 1e6
	p.Health = p.MaxHealth
	boss.TargetPlayerID = p.ID

	w := &World{
		ZoneType:      1,
		ZoneID:        "arena_test",
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{boss},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
	}
	w.InitGateStates()
	// The room seals when the fight starts, as in live gameflow.
	for i := range w.Level.Gates {
		if w.Level.Gates[i].ID == testAcerasGateID {
			w.GateStates[testAcerasGateID] = true
		}
	}
	w.RebuildObstacles()
	brain := enemyai.NewBrain(def, boss, w.AbilityEngine)
	brain.BoundsMinX = w.Level.EnemyBoundsMinX
	brain.BoundsMaxX = w.Level.EnemyBoundsMaxX
	brain.BoundsMinZ = w.Level.EnemyBoundsMinZ
	brain.BoundsMaxZ = w.Level.EnemyBoundsMaxZ
	w.Brains = []enemyai.BrainTicker{brain}

	ai := AISystem{}
	var stalker *entity.Enemy
	stalkerSpawnTick := -1
	rakeHits := 0
	var spawnPos entity.Vec3
	moved := float32(0)

	for tick := range 800 { // 40s at 20Hz
		w.TickNum = uint32(tick)
		w.DamageEvents = w.DamageEvents[:0]
		ai.Tick(w, 0.05)

		if stalker == nil {
			for _, e := range w.Enemies {
				if e.DefName == "aceras_stalker" && e.Alive {
					stalker = e
					stalkerSpawnTick = tick
					spawnPos = e.Position
					break
				}
			}
		} else {
			if d := stalker.Position.Flat().DistanceTo(spawnPos.Flat()); d > moved {
				moved = d
			}
			for _, ev := range w.DamageEvents {
				if ev.SourceType == combat.SourceEnemyAddMelee && ev.TargetPeerID == p.ID {
					rakeHits++
				}
			}
			// A real player repositions constantly: walk away from the stalker
			// at run speed whenever it gets close, staying inside the room.
			dir := p.Position.Sub(stalker.Position).Flat()
			if dir.Length() < 6 && dir.Length() > 0.01 {
				p.Position = p.Position.Add(dir.Normalized().Scale(6.0 * 0.05))
				p.Position.X = entity.Clamp(p.Position.X, -12, 12)
				p.Position.Z = entity.Clamp(p.Position.Z, -70, -44)
			}
		}
	}

	if stalker == nil {
		t.Fatal("the General never summoned a stalker in 40s (summon opener is 8s)")
	}
	t.Logf("stalker spawned at tick %d pos %v; moved %.1fm; rake hits on player: %d",
		stalkerSpawnTick, spawnPos, moved, rakeHits)
	if moved < 1.0 {
		t.Errorf("stalker never moved (max displacement %.2fm) — frozen add", moved)
	}
	if rakeHits == 0 {
		t.Error("stalker never landed talon_rake on an adjacent player — frozen add")
	}
}
