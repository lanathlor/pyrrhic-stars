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
			wantAmount:   50,
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
			wantAmount:   30,
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
			wantAmount:   40,
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
			wantAmount:   25,
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
			wantAmount:   10, // capped at MaxHealth
			wantSource:   combat.SourcePlayerHeal,
		},
		{
			name: "heal on full HP returns nil",
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
			wantNil:      true,
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
				caster.GearStats.Identity = 100 // +100% heal
				target := newHealer(2)
				target.Health = 20
				target.MaxHealth = 170
				return map[uint16]*entity.Player{1: caster, 2: target}
			},
			targetPeerID: 2,
			wantNil:      false,
			wantTargetID: 2,
			wantAmount:   100, // 50 * (1 + 100/100) = 100
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
			if result.SourceID != tt.casterID {
				t.Errorf("SourceID = %d, want %d", result.SourceID, tt.casterID)
			}
			if result.SourceType != tt.wantSource {
				t.Errorf("SourceType = %d, want %d", result.SourceType, tt.wantSource)
			}
		})
	}
}
