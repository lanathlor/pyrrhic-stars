package zone

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

// TestZone_SpawnedStalkerChasesAndRakes reproduces the live Aceras General
// fight through the full zone tick pipeline (inputs, AI, combat, physics,
// gameflow, gates): a solo vanguard engages the General; the first stalker
// wave must chase the player and land talon_rake hits. Live run 2026-07-14
// observed stalkers that neither moved nor attacked.
func TestZone_SpawnedStalkerChasesAndRakes(t *testing.T) {
	z := New("test_arena", testArenaLevel(t), nil)

	var general *entity.Enemy
	for _, e := range z.world.Enemies {
		if e.BossNum == 2 {
			general = e
		}
		e.Alive = true
	}
	if general == nil {
		t.Fatal("no boss_num 2 enemy in arena level")
	}

	peerID := uint16(1)
	p := entity.NewPlayer(peerID, entity.ClassVanguard)
	p.MaxHealth = 1e6
	p.Health = p.MaxHealth
	p.Position = general.Position.Add(entity.Vec3{Z: 6})
	p.Alive = true
	z.world.Players[peerID] = p
	z.world.Clients[peerID] = &Client{
		PeerID:   peerID,
		Username: testPlayerName,
		Send:     func([]byte) {},
		SendUDP:  func([]byte) {},
		HasUDP:   func() bool { return true },
	}

	var stalker *entity.Enemy
	var spawnPos entity.Vec3
	moved := float32(0)
	stalkerDamage := float32(0)
	prevHP := p.Health

	for range 1200 { // 60s at 20Hz
		z.processTick()

		if stalker == nil {
			for _, e := range z.world.Enemies {
				if e.DefName == "aceras_stalker" && e.Alive {
					stalker = e
					spawnPos = e.Position
					prevHP = p.Health
					break
				}
			}
			continue
		}
		if d := stalker.Position.Flat().DistanceTo(spawnPos.Flat()); d > moved {
			moved = d
		}
		// Attribute HP loss while adjacent to the stalker (the boss kites away
		// during its evasive phase, so at melee range the rake is the source).
		if p.Health < prevHP && stalker.Position.Flat().DistanceTo(p.Position.Flat()) < 5 {
			stalkerDamage += prevHP - p.Health
		}
		prevHP = p.Health

		// The player repositions like a real one: run from the stalker when
		// it closes, staying inside the General's room.
		if stalker.Alive {
			dir := p.Position.Sub(stalker.Position).Flat()
			if dir.Length() < 6 && dir.Length() > 0.01 {
				p.Position = p.Position.Add(dir.Normalized().Scale(6.0 * 0.05))
				p.Position.X = entity.Clamp(p.Position.X, -12, 12)
				p.Position.Z = entity.Clamp(p.Position.Z, -70, -44)
				p.Position.Y = float32(math.Max(float64(p.Position.Y), -4.0))
			}
		}
	}

	if stalker == nil {
		t.Fatal("the General never summoned a stalker in 60s of zone ticks")
	}
	t.Logf("stalker spawned at %v; moved %.1fm; damage while adjacent: %.0f", spawnPos, moved, stalkerDamage)
	if moved < 1.0 {
		t.Errorf("stalker never moved (max displacement %.2fm) — frozen add", moved)
	}
	if stalkerDamage == 0 {
		t.Error("stalker never damaged the player — frozen add")
	}
}
