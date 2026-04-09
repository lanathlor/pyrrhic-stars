package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// HitType identifies how a Blade Dancer spell resolves damage.
type HitType uint8

const (
	HitNone         HitType = 0 // self-buff only, no damage
	HitSingleTarget HitType = 1 // hitscan single target
	HitAoECircle    HitType = 2 // circle AoE around caster
	HitAoECone      HitType = 3 // frontal cone
)

// BDSpellDef defines one of the 20 Blade Dancer transition spells.
type BDSpellDef struct {
	Name      string
	OriginCfg int
	DestCfg   int
	Duration  float32 // cast time (GCD handles pacing)

	// Damage
	Hit        HitType
	Damage     float32
	Radius     float32 // for AoE
	ArcDegrees float32 // for cone

	// Buffs (applied to caster)
	ShieldHP   float32 // if >0, grants temporary shield
	DRFactor   float32 // if >0, damage reduction (e.g. 0.8 = 20% DR)
	DRDuration float32 // duration of DR buff

	// Debuffs (applied to hit targets)
	// These are simplified for Phase 0 — full implementation later
}

// destSlotToConfig maps a spell slot (0-3) within a given origin config
// to the destination config, skipping the origin config itself.
// For origin config C, slot i maps to the i-th config in {0,1,2,3,4}\{C}.
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
	return 0 // should not reach
}

// bdSpellTable holds all 20 Blade Dancer spell definitions.
// Indexed by action_id - ActionBDSpellBase (0-19).
// Layout: spells 0-3 = from Orbit, 4-7 = from Fan, 8-11 = from Lance,
//
//	12-15 = from Scatter, 16-19 = from Crown.
//
// Within each group, slot 0-3 maps to the 4 destination configs (skipping self).
var bdSpellTable = [20]BDSpellDef{
	// === From Orbit (Defense) ===
	// Slot 0: Orbit → Fan — Shielded Sweep
	{Name: "Shielded Sweep", OriginCfg: 0, DestCfg: 1, Duration: 0.4,
		Hit: HitAoECone, Damage: 30.0, Radius: 4.0, ArcDegrees: 120.0,
		DRFactor: 0.85, DRDuration: 2.0},
	// Slot 1: Orbit → Lance — Guarded Thrust
	{Name: "Guarded Thrust", OriginCfg: 0, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 35.0,
		ShieldHP: 8.0},
	// Slot 2: Orbit → Scatter — Protected Scatter
	{Name: "Protected Scatter", OriginCfg: 0, DestCfg: 3, Duration: 0.4,
		Hit: HitAoECircle, Damage: 15.0, Radius: 6.0,
		DRFactor: 0.9, DRDuration: 1.5},
	// Slot 3: Orbit → Crown — Fortified Command
	{Name: "Fortified Command", OriginCfg: 0, DestCfg: 4, Duration: 0.5,
		Hit: HitAoECircle, Damage: 10.0, Radius: 5.0,
		DRFactor: 0.8, DRDuration: 2.0},

	// === From Fan (AoE Damage) ===
	// Slot 0: Fan → Orbit — Reaping Guard
	{Name: "Reaping Guard", OriginCfg: 1, DestCfg: 0, Duration: 0.4,
		Hit: HitAoECircle, Damage: 15.0, Radius: 3.0,
		ShieldHP: 12.0},
	// Slot 1: Fan → Lance — Cleaving Pierce
	{Name: "Cleaving Pierce", OriginCfg: 1, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 45.0},
	// Slot 2: Fan → Scatter — Slashing Spread
	{Name: "Slashing Spread", OriginCfg: 1, DestCfg: 3, Duration: 0.4,
		Hit: HitAoECircle, Damage: 25.0, Radius: 5.0},
	// Slot 3: Fan → Crown — Sweeping Hex
	{Name: "Sweeping Hex", OriginCfg: 1, DestCfg: 4, Duration: 0.5,
		Hit: HitAoECone, Damage: 20.0, Radius: 5.0, ArcDegrees: 120.0},

	// === From Lance (Single-target Damage) ===
	// Slot 0: Lance → Orbit — Piercing Barrier
	{Name: "Piercing Barrier", OriginCfg: 2, DestCfg: 0, Duration: 0.4,
		Hit: HitSingleTarget, Damage: 25.0,
		ShieldHP: 15.0},
	// Slot 1: Lance → Fan — Focused Slash
	{Name: "Focused Slash", OriginCfg: 2, DestCfg: 1, Duration: 0.3,
		Hit: HitAoECone, Damage: 35.0, Radius: 4.0, ArcDegrees: 90.0},
	// Slot 2: Lance → Scatter — Targeted Spread
	{Name: "Targeted Spread", OriginCfg: 2, DestCfg: 3, Duration: 0.4,
		Hit: HitSingleTarget, Damage: 30.0},
	// Slot 3: Lance → Crown — Pinning Strike
	{Name: "Pinning Strike", OriginCfg: 2, DestCfg: 4, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 35.0},

	// === From Scatter (Multi-target DoT) ===
	// Slot 0: Scatter → Orbit — Dispersed Shield
	{Name: "Dispersed Shield", OriginCfg: 3, DestCfg: 0, Duration: 0.5,
		Hit: HitNone,
		ShieldHP: 18.0, DRFactor: 0.85, DRDuration: 2.0},
	// Slot 1: Scatter → Fan — Rain of Blades
	{Name: "Rain of Blades", OriginCfg: 3, DestCfg: 1, Duration: 0.4,
		Hit: HitAoECircle, Damage: 35.0, Radius: 5.0},
	// Slot 2: Scatter → Lance — Converging Strike
	{Name: "Converging Strike", OriginCfg: 3, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 50.0},
	// Slot 3: Scatter → Crown — Chaos Bind
	{Name: "Chaos Bind", OriginCfg: 3, DestCfg: 4, Duration: 0.5,
		Hit: HitAoECircle, Damage: 15.0, Radius: 5.0},

	// === From Crown (Utility/Control) ===
	// Slot 0: Crown → Orbit — Commanding Ward
	{Name: "Commanding Ward", OriginCfg: 4, DestCfg: 0, Duration: 0.5,
		Hit: HitNone,
		ShieldHP: 20.0},
	// Slot 1: Crown → Fan — Royal Cleave
	{Name: "Royal Cleave", OriginCfg: 4, DestCfg: 1, Duration: 0.3,
		Hit: HitAoECone, Damage: 30.0, Radius: 5.0, ArcDegrees: 120.0},
	// Slot 2: Crown → Lance — Decree Strike
	{Name: "Decree Strike", OriginCfg: 4, DestCfg: 2, Duration: 0.3,
		Hit: HitSingleTarget, Damage: 40.0},
	// Slot 3: Crown → Scatter — Sovereign Scatter
	{Name: "Sovereign Scatter", OriginCfg: 4, DestCfg: 3, Duration: 0.4,
		Hit: HitAoECircle, Damage: 20.0, Radius: 6.0},
}

// resolveBladeDancerSpell executes one of the 20 Blade Dancer transition spells.
func resolveBladeDancerSpell(w *World, p *entity.Player, peerID uint16, spellIdx int) {
	if spellIdx < 0 || spellIdx >= 20 {
		return
	}
	spell := &bdSpellTable[spellIdx]

	// Validate origin config matches
	if spell.OriginCfg != p.Config {
		return
	}

	// Resolve damage
	switch spell.Hit {
	case HitSingleTarget:
		evt, hitEnemy := combat.ResolvePlayerAttackOnEnemies(p, w.Enemies, w.Level.Obstacles)
		if evt != nil {
			evt.SourcePeerID = peerID
			evt.Amount = spell.Damage
			// Re-apply damage with correct amount (the resolve function used old BD damage)
			// Actually, we need to apply the spell-specific damage directly
			w.DamageEvents = append(w.DamageEvents, *evt)
			if hitEnemy != nil {
				hitEnemy.AddThreat(peerID, evt.Amount)
				w.AggroEnemy(hitEnemy, peerID)
			}
		}
	case HitAoECircle:
		shape := combat.AoEShape{Type: combat.AoECircle, Radius: spell.Radius, Damage: spell.Damage}
		events := combat.ResolvePlayerAoEOnEnemies(p, w.Enemies, w.Level.Obstacles, shape)
		for _, evt := range events {
			w.DamageEvents = append(w.DamageEvents, evt)
			for _, e := range w.Enemies {
				if e != nil && e.ID == evt.TargetPeerID {
					e.AddThreat(peerID, evt.Amount)
					w.AggroEnemy(e, peerID)
					break
				}
			}
		}
	case HitAoECone:
		shape := combat.AoEShape{Type: combat.AoECone, Radius: spell.Radius, ArcDegrees: spell.ArcDegrees, Damage: spell.Damage}
		events := combat.ResolvePlayerAoEOnEnemies(p, w.Enemies, w.Level.Obstacles, shape)
		for _, evt := range events {
			w.DamageEvents = append(w.DamageEvents, evt)
			for _, e := range w.Enemies {
				if e != nil && e.ID == evt.TargetPeerID {
					e.AddThreat(peerID, evt.Amount)
					w.AggroEnemy(e, peerID)
					break
				}
			}
		}
	case HitNone:
		// Self-buff only, no damage
	}

	// Apply caster buffs (shield caps at 25, doesn't stack infinitely)
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
