package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

func newHealer(id uint16) *entity.Player {
	p := entity.NewPlayer(id, entity.ClassArcanotechnicien)
	return p
}

func TestResolveHeal(t *testing.T) {
	tests := []struct {
		name         string
		def          *AbilityDef
		casterID     uint16
		allies       func() map[uint16]*entity.Player
		targetPeerID uint16
		wantNil      bool
		wantTargetID uint16
		wantAmount   float32
		wantOverheal float32
		wantSource   uint8
	}{
		{
			name: "HitAllyTarget heals valid target",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 50,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				target := newHealer(2)
				target.Health = 100
				target.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   57.5, // 50 * 1.15 (Sympathetic Field: Harmonist caster, target at same pos)
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "HitAllyTarget falls back to self when target invalid",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 30,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Health = 100
				caster.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster}
			},
			targetPeerID: 99, // does not exist
			wantNil:      false,
			wantTargetID: 1,
			wantAmount:   34.5, // 30 * 1.15 (Sympathetic Field: self-heal, dist=0)
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "HitAllyLowestHP picks lowest HP ally",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyLowestHP},
				BaseHeal: 40,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Health = 150
				caster.MaxHealth = 170
				a := newHealer(2)
				a.Health = 80
				a.MaxHealth = 170
				b := newHealer(3)
				b.Health = 50
				b.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: a, 3: b}
			},
			targetPeerID: 0,
			wantNil:      false,
			wantTargetID: 3,
			wantAmount:   46, // 40 * 1.15 (Sympathetic Field: Harmonist, same pos)
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "HitAllyRandom picks an injured ally",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyRandom},
				BaseHeal: 25,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Health = caster.MaxHealth // full HP
				injured := newHealer(2)
				injured.Health = 100
				injured.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: injured}
			},
			targetPeerID: 0,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   28.75, // 25 * 1.15 (Sympathetic Field: Harmonist, same pos)
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "heal does not exceed MaxHealth",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 100,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				target := newHealer(2)
				target.Health = 160
				target.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   10,    // capped at MaxHealth
			wantOverheal: 105.0, // 100 * 1.15 (SF) - 10 actual = 105
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "heal on full HP returns overheal",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 50,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Health = caster.MaxHealth
				target := newHealer(2)
				target.Health = target.MaxHealth // already full
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   0,
			wantOverheal: 57.5, // 50 * 1.15 (Sympathetic Field)
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "zero BaseHeal returns nil",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 0,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				target := newHealer(2)
				target.Health = 50
				target.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      true,
		},
		{
			name: "non-ally HitType returns nil",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitMeleeArc},
				BaseHeal: 50,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				return map[uint16]*entity.Player{1: caster}
			},
			targetPeerID: 0,
			wantNil:      true,
		},
		{
			name: "Identity stat scales heal amount",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 50,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.GearStats.Identity = 100 // +100% heal, also scales SF radius to 12
				target := newHealer(2)
				target.Health = 20
				target.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   115, // 50 * 2.0 (identity) * 1.15 (Sympathetic Field) = 115
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "Sympathetic Field amplifies heal when target is in range",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 100,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
				target := newHealer(2)
				target.Position = entity.Vec3{X: 3, Y: 0, Z: 4} // dist = 5, within radius 8
				target.Health = 20
				target.MaxHealth = 500
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   115, // 100 * 1.15
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "Sympathetic Field does not amplify when target is out of range",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 100,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
				target := newHealer(2)
				target.Position = entity.Vec3{X: 6, Y: 0, Z: 7} // dist ~9.22, outside radius 8
				target.Health = 20
				target.MaxHealth = 500
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   100, // no amplification
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "Sympathetic Field radius scales with Identity",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 100,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := newHealer(1)
				caster.GearStats.Identity = 100 // radius = 8 * (1 + 100/200) = 12
				caster.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
				target := newHealer(2)
				target.Position = entity.Vec3{X: 7, Y: 0, Z: 8} // dist ~10.63, within 12
				target.Health = 20
				target.MaxHealth = 1000
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   230, // 100 * (1 + 100/100) * 1.15 = 230
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "non-Harmonist caster does not amplify",
			def: &AbilityDef{
				Hit:      HitDef{Type: HitAllyTarget},
				BaseHeal: 100,
			},
			casterID: 1,
			allies: func() map[uint16]*entity.Player {
				caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "destroyer")
				caster.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
				target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "destroyer")
				target.Position = entity.Vec3{X: 1, Y: 0, Z: 0} // very close
				target.Health = 20
				target.MaxHealth = 500
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   100, // no amplification (not Harmonist)
			wantSource:   combat.SourcePlayerHeal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allies := tt.allies()
			caster := allies[tt.casterID]
			result := resolveHeal(tt.def, caster, allies, tt.targetPeerID)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got HealResult{TargetID: %d, Amount: %f}", result.TargetID, result.Amount)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil HealResult, got nil")
			}
			if result.TargetID != tt.wantTargetID {
				t.Errorf("TargetID = %d, want %d", result.TargetID, tt.wantTargetID)
			}
			if math.Abs(float64(result.Amount-tt.wantAmount)) > 0.01 {
				t.Errorf("Amount = %f, want %f", result.Amount, tt.wantAmount)
			}
			if math.Abs(float64(result.Overheal-tt.wantOverheal)) > 0.01 {
				t.Errorf("Overheal = %f, want %f", result.Overheal, tt.wantOverheal)
			}
			if result.SourceID != tt.casterID {
				t.Errorf("SourceID = %d, want %d", result.SourceID, tt.casterID)
			}
			if result.SourceType != tt.wantSource {
				t.Errorf("SourceType = %d, want %d", result.SourceType, tt.wantSource)
			}
		})
	}
}

func TestHitAllyRandom_UsesRNG(t *testing.T) {
	// The old code used caster.ID % len(injured), which meant two casters
	// with IDs that are equal mod N always pick the same target.
	// With caster ID=3 and 3 injured allies, 3%3=0 always picks index 0.
	// With caster ID=6, 6%3=0 also always picks index 0.
	// Both should sometimes differ if using actual randomness.
	//
	// Since resolveHealTarget builds its injured slice from a map (random
	// iteration order), the modulus bug is partially masked. But the
	// fundamental contract is wrong: selection depends on caster.ID, not RNG.
	// After the fix, the function should accept an RNG and use it.
	def := &AbilityDef{Hit: HitDef{Type: HitAllyRandom}}

	a1 := entity.NewPlayer(1, entity.ClassGunner)
	a1.Health = 50
	a1.MaxHealth = 100
	a2 := entity.NewPlayer(2, entity.ClassGunner)
	a2.Health = 50
	a2.MaxHealth = 100

	// Only 2 injured allies: caster (ID=4) is at full HP, so not in injured list.
	caster := entity.NewPlayer(4, entity.ClassArcanotechnicien)
	caster.Health = 100
	caster.MaxHealth = 100
	allies := map[uint16]*entity.Player{1: a1, 2: a2, 4: caster}

	// With old code: 4 % 2 = 0 => always picks index 0 (same ally every time)
	// After fix: should vary with RNG across calls
	seen := map[uint16]bool{}
	for range 100 {
		target := resolveHealTarget(def, caster, allies, 0)
		if target != nil {
			seen[target.ID] = true
		}
	}
	// With 2 injured allies and 100 calls, both should appear if random
	if len(seen) < 2 {
		t.Errorf("HitAllyRandom always picks the same target (saw %v); should vary", seen)
	}
}
