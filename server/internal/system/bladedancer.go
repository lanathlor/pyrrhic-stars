package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// HitType identifies how a Blade Dancer spell resolves damage.
type HitType uint8

const (
	HitNone            HitType = 0 // self-buff only, no damage
	HitSingleTarget    HitType = 1 // hitscan single target
	HitAoECircle       HitType = 2 // circle AoE around caster (player-centered)
	HitAoECircleTarget HitType = 4 // circle AoE centered on hitscan target
	HitNearestN        HitType = 5 // nearest N in-combat enemies (proximity, no geometry)
)

// BDSpellDef defines one of the 20 Blade Dancer transition spells.
type BDSpellDef struct {
	Name      string
	OriginCfg int
	DestCfg   int
	Duration  float32

	// Damage
	Hit         HitType
	Damage      float32
	Radius      float32 // for AoE circle
	TargetCount int     // for HitNearestN

	// Buffs (applied to caster)
	ShieldHP   float32
	DRFactor   float32
	DRDuration float32

	// DoT (applied to hit targets)
	DoTDamage   float32
	DoTDuration float32
	DoTInterval float32
}

// BDDoT tracks an active damage-over-time effect on an enemy.
type BDDoT struct {
	EnemyID    uint16
	SourcePeer uint16
	Damage     float32
	Remaining  float32
	Interval   float32
	TickTimer  float32
}

// destSlotToConfig maps a spell slot (0-3) within a given origin config
// to the destination config, skipping the origin config itself.
func destSlotToConfig(originCfg int, slot int) int {
	idx := 0
	for c := 0; c < 5; c++ {
		if c == originCfg {
			continue
		}
		if idx == slot {
			return c
		}
		idx++
	}
	return 0
}

// bdSpellTable holds all 20 Blade Dancer spell definitions.
// Indexed by action_id - ActionBDSpellBase (0-19).
// PC = player-centered, TC = target-centered, ST = single-target, N(x) = nearest x
var bdSpellTable = [20]BDSpellDef{
	// === From Orbit (Defense) ===
	// 0: Orbit → Fan — Shielded Sweep [PC circle]
	{Name: "Shielded Sweep", OriginCfg: 0, DestCfg: 1, Duration: 0.4,
		Hit: HitAoECircle, Damage: 8.0, Radius: 4.0,
		DRFactor: 0.85, DRDuration: 2.0},
	// 1: Orbit → Lance — Guarded Thrust [ST]
	{Name: "Guarded Thrust", OriginCfg: 0, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 25.0,
		ShieldHP: 8.0},
	// 2: Orbit → Scatter — Protected Scatter [N(3)]
	{Name: "Protected Scatter", OriginCfg: 0, DestCfg: 3, Duration: 0.4,
		Hit: HitNearestN, Damage: 5.0, TargetCount: 3,
		DRFactor: 0.9, DRDuration: 1.5,
		DoTDamage: 1.5, DoTDuration: 12.0, DoTInterval: 1.0},
	// 3: Orbit → Crown — Fortified Command [TC circle]
	{Name: "Fortified Command", OriginCfg: 0, DestCfg: 4, Duration: 0.5,
		Hit: HitAoECircleTarget, Damage: 5.0, Radius: 5.0,
		DRFactor: 0.8, DRDuration: 2.0},

	// === From Fan (AoE Damage) ===
	// 4: Fan → Orbit — Reaping Guard [PC circle]
	{Name: "Reaping Guard", OriginCfg: 1, DestCfg: 0, Duration: 0.4,
		Hit: HitAoECircle, Damage: 8.0, Radius: 3.0,
		ShieldHP: 12.0},
	// 5: Fan → Lance — Cleaving Pierce [ST]
	{Name: "Cleaving Pierce", OriginCfg: 1, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 30.0},
	// 6: Fan → Scatter — Slashing Spread [TC circle]
	{Name: "Slashing Spread", OriginCfg: 1, DestCfg: 3, Duration: 0.4,
		Hit: HitAoECircleTarget, Damage: 8.0, Radius: 5.0,
		DoTDamage: 1.5, DoTDuration: 10.0, DoTInterval: 1.0},
	// 7: Fan → Crown — Sweeping Hex [TC circle]
	{Name: "Sweeping Hex", OriginCfg: 1, DestCfg: 4, Duration: 0.5,
		Hit: HitAoECircleTarget, Damage: 10.0, Radius: 5.0},

	// === From Lance (Single-target Damage) ===
	// 8: Lance → Orbit — Piercing Barrier [ST]
	{Name: "Piercing Barrier", OriginCfg: 2, DestCfg: 0, Duration: 0.4,
		Hit: HitSingleTarget, Damage: 18.0,
		ShieldHP: 15.0},
	// 9: Lance → Fan — Focused Slash [TC circle]
	{Name: "Focused Slash", OriginCfg: 2, DestCfg: 1, Duration: 0.3,
		Hit: HitAoECircleTarget, Damage: 15.0, Radius: 4.0},
	// 10: Lance → Scatter — Targeted Spread [ST + long DoT]
	{Name: "Targeted Spread", OriginCfg: 2, DestCfg: 3, Duration: 0.4,
		Hit: HitSingleTarget, Damage: 12.0,
		DoTDamage: 2.0, DoTDuration: 15.0, DoTInterval: 1.0},
	// 11: Lance → Crown — Pinning Strike [ST]
	{Name: "Pinning Strike", OriginCfg: 2, DestCfg: 4, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 25.0},

	// === From Scatter (Multi-target DoT) ===
	// 12: Scatter → Orbit — Dispersed Shield [self-buff]
	{Name: "Dispersed Shield", OriginCfg: 3, DestCfg: 0, Duration: 0.5,
		Hit: HitNone,
		ShieldHP: 18.0, DRFactor: 0.85, DRDuration: 2.0},
	// 13: Scatter → Fan — Rain of Blades [TC circle + DoT]
	{Name: "Rain of Blades", OriginCfg: 3, DestCfg: 1, Duration: 0.4,
		Hit: HitAoECircleTarget, Damage: 15.0, Radius: 5.0,
		DoTDamage: 1.0, DoTDuration: 10.0, DoTInterval: 1.0},
	// 14: Scatter → Lance — Converging Strike [ST + DoT]
	{Name: "Converging Strike", OriginCfg: 3, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 32.0,
		DoTDamage: 1.5, DoTDuration: 10.0, DoTInterval: 1.0},
	// 15: Scatter → Crown — Chaos Bind [N(4)]
	{Name: "Chaos Bind", OriginCfg: 3, DestCfg: 4, Duration: 0.5,
		Hit: HitNearestN, Damage: 8.0, TargetCount: 4},

	// === From Crown (Utility/Control) ===
	// 16: Crown → Orbit — Commanding Ward [self-buff]
	{Name: "Commanding Ward", OriginCfg: 4, DestCfg: 0, Duration: 0.5,
		Hit: HitNone,
		ShieldHP: 20.0},
	// 17: Crown → Fan — Royal Cleave [PC circle]
	{Name: "Royal Cleave", OriginCfg: 4, DestCfg: 1, Duration: 0.3,
		Hit: HitAoECircle, Damage: 12.0, Radius: 5.0},
	// 18: Crown → Lance — Decree Strike [ST]
	{Name: "Decree Strike", OriginCfg: 4, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 28.0},
	// 19: Crown → Scatter — Sovereign Scatter [N(3) + DoT]
	{Name: "Sovereign Scatter", OriginCfg: 4, DestCfg: 3, Duration: 0.4,
		Hit: HitNearestN, Damage: 5.0, TargetCount: 3,
		DoTDamage: 1.5, DoTDuration: 12.0, DoTInterval: 1.0},
}

// findHitscanTarget finds the nearest enemy hit by the player's hitscan aim.
func findHitscanTarget(p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle) *entity.Enemy {
	origin := p.EyePosition()
	direction := p.AimDirection()
	var best *entity.Enemy
	bestDistSq := float32(1e18)
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		targetCenter := e.Position.Add(entity.Vec3{Y: 1.0})
		if !combat.CheckHitscan(origin, direction, targetCenter, 1.2, 20.0, obstacles) {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = e
		}
	}
	return best
}

// resolveBladeDancerSpell executes one of the 20 Blade Dancer transition spells.
func resolveBladeDancerSpell(w *World, p *entity.Player, peerID uint16, spellIdx int) {
	if spellIdx < 0 || spellIdx >= 20 {
		return
	}
	spell := &bdSpellTable[spellIdx]

	if spell.OriginCfg != p.Config {
		return
	}

	// Resolve damage — collect hit enemy IDs for DoT application
	var hitEnemyIDs []uint16

	switch spell.Hit {
	case HitSingleTarget:
		hitEnemy := findHitscanTarget(p, w.Enemies, w.Level.Obstacles)
		if hitEnemy != nil {
			dealt, _ := hitEnemy.ApplyDamage(spell.Damage)
			if dealt > 0 {
				hitDir := hitEnemy.Position.Sub(p.Position)
				if hitDir.LengthSq() > 0.01 {
					hitDir = hitDir.Normalized()
				}
				w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
					TargetPeerID: hitEnemy.ID,
					SourcePeerID: peerID,
					Amount:       dealt,
					HitPos:       p.Position.Add(hitDir),
					SourceType:   combat.SourcePlayerAttack,
				})
				hitEnemy.AddThreat(peerID, dealt)
				w.AggroEnemy(hitEnemy, peerID)
				hitEnemyIDs = append(hitEnemyIDs, hitEnemy.ID)
			}
		}

	case HitAoECircle:
		// Player-centered circle
		shape := combat.AoEShape{Type: combat.AoECircle, Radius: spell.Radius, Damage: spell.Damage}
		events := combat.ResolvePlayerAoEOnEnemies(p, w.Enemies, w.Level.Obstacles, shape)
		for _, evt := range events {
			w.DamageEvents = append(w.DamageEvents, evt)
			hitEnemyIDs = append(hitEnemyIDs, evt.TargetPeerID)
			for _, e := range w.Enemies {
				if e != nil && e.ID == evt.TargetPeerID {
					e.AddThreat(peerID, evt.Amount)
					w.AggroEnemy(e, peerID)
					break
				}
			}
		}

	case HitAoECircleTarget:
		// Target-centered circle: find hitscan target, AoE around that position
		target := findHitscanTarget(p, w.Enemies, w.Level.Obstacles)
		if target != nil {
			shape := combat.AoEShape{Radius: spell.Radius, Damage: spell.Damage}
			events := combat.ResolveAoEAtPosition(target.Position, peerID, w.Enemies, w.Level.Obstacles, shape)
			for _, evt := range events {
				w.DamageEvents = append(w.DamageEvents, evt)
				hitEnemyIDs = append(hitEnemyIDs, evt.TargetPeerID)
				for _, e := range w.Enemies {
					if e != nil && e.ID == evt.TargetPeerID {
						e.AddThreat(peerID, evt.Amount)
						w.AggroEnemy(e, peerID)
						break
					}
				}
			}
		}

	case HitNearestN:
		events := combat.ResolveNearestNEnemies(p, w.Enemies, w.Level.Obstacles, spell.TargetCount, spell.Damage)
		for _, evt := range events {
			w.DamageEvents = append(w.DamageEvents, evt)
			hitEnemyIDs = append(hitEnemyIDs, evt.TargetPeerID)
			for _, e := range w.Enemies {
				if e != nil && e.ID == evt.TargetPeerID {
					e.AddThreat(peerID, evt.Amount)
					w.AggroEnemy(e, peerID)
					break
				}
			}
		}

	case HitNone:
		// Self-buff only
	}

	// Apply DoTs to hit enemies
	if spell.DoTDamage > 0 && len(hitEnemyIDs) > 0 {
		for _, eid := range hitEnemyIDs {
			w.BDDoTs = append(w.BDDoTs, BDDoT{
				EnemyID:    eid,
				SourcePeer: peerID,
				Damage:     spell.DoTDamage,
				Remaining:  spell.DoTDuration,
				Interval:   spell.DoTInterval,
				TickTimer:  spell.DoTInterval,
			})
		}
	}

	// Apply caster buffs (shield caps at 25)
	if spell.ShieldHP > 0 {
		p.BDShieldHP += spell.ShieldHP
		if p.BDShieldHP > 25.0 {
			p.BDShieldHP = 25.0
		}
	}
	if spell.DRFactor > 0 {
		p.BDDRFactor = spell.DRFactor
		p.BDDRTimer = spell.DRDuration
	}

	// Transition config
	p.Config = spell.DestCfg
	p.GCDTimer = 0.5
	p.State = entity.PlayerStateAttack
}
