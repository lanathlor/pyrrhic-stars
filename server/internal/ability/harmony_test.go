package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func newHarmonist(id uint16) *entity.Player {
	p := entity.NewPlayerWithSpec(id, entity.ClassArcanotechnicien, "harmonist")
	return p
}

func TestCheckHarmony(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*entity.Player, uint16)
		deliveries []entity.DeliveryMethod
		wantBonus  []float32 // expected return per delivery call
	}{
		{
			name: "first heal on target returns zero",
			setup: func() (*entity.Player, uint16) {
				return newHarmonist(1), 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect},
			wantBonus:  []float32{0},
		},
		{
			name: "same delivery method twice returns zero",
			setup: func() (*entity.Player, uint16) {
				return newHarmonist(1), 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryDirect},
			wantBonus:  []float32{0, 0},
		},
		{
			name: "different delivery method triggers Harmony",
			setup: func() (*entity.Player, uint16) {
				return newHarmonist(1), 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryBeam},
			wantBonus:  []float32{0, 20},
		},
		{
			name: "rotation Direct Beam Zone Direct procs on each switch",
			setup: func() (*entity.Player, uint16) {
				return newHarmonist(1), 2
			},
			deliveries: []entity.DeliveryMethod{
				entity.DeliveryDirect,
				entity.DeliveryBeam,
				entity.DeliveryZone,
				entity.DeliveryDirect,
			},
			wantBonus: []float32{0, 20, 20, 20},
		},
		{
			name: "Mastery scaling affects bonus amount",
			setup: func() (*entity.Player, uint16) {
				p := newHarmonist(1)
				p.GearStats.Mastery = 50 // 50 Mastery = 1.5x multiplier
				return p, 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryBeam},
			wantBonus:  []float32{0, 30}, // 20 * (1 + 50/100) = 30
		},
		{
			name: "high Mastery scaling",
			setup: func() (*entity.Player, uint16) {
				p := newHarmonist(1)
				p.GearStats.Mastery = 100 // 100 Mastery = 2x multiplier
				return p, 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryZone},
			wantBonus:  []float32{0, 40}, // 20 * (1 + 100/100) = 40
		},
		{
			name: "non-Harmonist player returns zero",
			setup: func() (*entity.Player, uint16) {
				p := entity.NewPlayer(1, entity.ClassGunner)
				return p, 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryBeam},
			wantBonus:  []float32{0, 0},
		},
		{
			name: "non-Harmonist Arcanotechnicien returns zero",
			setup: func() (*entity.Player, uint16) {
				p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "destroyer")
				return p, 2
			},
			deliveries: []entity.DeliveryMethod{entity.DeliveryDirect, entity.DeliveryBeam},
			wantBonus:  []float32{0, 0},
		},
		{
			name: "separate targets tracked independently",
			setup: func() (*entity.Player, uint16) {
				return newHarmonist(1), 0 // targetID managed per call below
			},
			deliveries: nil, // custom sequence below
			wantBonus:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healer, targetID := tt.setup()

			if tt.name == "separate targets tracked independently" {
				// Custom test: two targets, each tracked independently
				// Target 2: Direct (first = 0), then Beam (switch = 20)
				// Target 3: Beam (first = 0), then Beam (same = 0), then Direct (switch = 20)
				b1 := CheckHarmony(healer, 2, entity.DeliveryDirect)
				if b1 != 0 {
					t.Errorf("target 2 first heal = %f, want 0", b1)
				}
				b2 := CheckHarmony(healer, 3, entity.DeliveryBeam)
				if b2 != 0 {
					t.Errorf("target 3 first heal = %f, want 0", b2)
				}
				b3 := CheckHarmony(healer, 2, entity.DeliveryBeam)
				if math.Abs(float64(b3-20)) > 0.01 {
					t.Errorf("target 2 switch = %f, want 20", b3)
				}
				b4 := CheckHarmony(healer, 3, entity.DeliveryBeam)
				if b4 != 0 {
					t.Errorf("target 3 same = %f, want 0", b4)
				}
				b5 := CheckHarmony(healer, 3, entity.DeliveryDirect)
				if math.Abs(float64(b5-20)) > 0.01 {
					t.Errorf("target 3 switch = %f, want 20", b5)
				}
				return
			}

			for i, method := range tt.deliveries {
				got := CheckHarmony(healer, targetID, method)
				want := tt.wantBonus[i]
				if math.Abs(float64(got-want)) > 0.01 {
					t.Errorf("delivery[%d] (%d): bonus = %f, want %f", i, method, got, want)
				}
			}
		})
	}
}

func TestResolveHealHarmonyIntegration(t *testing.T) {
	tests := []struct {
		name              string
		setup             func() (*entity.Player, map[uint16]*entity.Player)
		defs              []*AbilityDef
		targetPeerID      uint16
		wantHarmonyProc   []bool
		wantHarmonyAmount []float32
		wantTotalHeal     []float32
	}{
		{
			name: "first heal no Harmony proc",
			setup: func() (*entity.Player, map[uint16]*entity.Player) {
				healer := newHarmonist(1)
				target := newHarmonist(2)
				target.Health = 50
				target.MaxHealth = 300
				return healer, map[uint16]*entity.Player{1: healer, 2: target}
			},
			defs: []*AbilityDef{
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryDirect)},
			},
			targetPeerID:      2,
			wantHarmonyProc:   []bool{false},
			wantHarmonyAmount: []float32{0},
			wantTotalHeal:     []float32{46}, // 40 * 1.15 (Sympathetic Field)
		},
		{
			name: "different delivery triggers Harmony bonus",
			setup: func() (*entity.Player, map[uint16]*entity.Player) {
				healer := newHarmonist(1)
				target := newHarmonist(2)
				target.Health = 50
				target.MaxHealth = 300
				return healer, map[uint16]*entity.Player{1: healer, 2: target}
			},
			defs: []*AbilityDef{
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryDirect)},
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryBeam)},
			},
			targetPeerID:      2,
			wantHarmonyProc:   []bool{false, true},
			wantHarmonyAmount: []float32{0, 20},
			wantTotalHeal:     []float32{46, 66}, // 40*1.15=46 base + 20 Harmony = 66
		},
		{
			name: "Harmony bonus capped by MaxHealth",
			setup: func() (*entity.Player, map[uint16]*entity.Player) {
				healer := newHarmonist(1)
				target := newHarmonist(2)
				target.Health = 100
				target.MaxHealth = 150
				return healer, map[uint16]*entity.Player{1: healer, 2: target}
			},
			defs: []*AbilityDef{
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryDirect)},
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryBeam)},
			},
			targetPeerID:    2,
			wantHarmonyProc: []bool{false, true},
			// After first heal: 100 + 46 = 146/150. Second: base 46 capped to 4, Harmony bonus = 0 (already full).
			wantHarmonyAmount: []float32{0, 0},
			wantTotalHeal:     []float32{46, 4}, // 40*1.15=46; then capped to 150-146=4
		},
		{
			name: "same delivery twice no Harmony",
			setup: func() (*entity.Player, map[uint16]*entity.Player) {
				healer := newHarmonist(1)
				target := newHarmonist(2)
				target.Health = 50
				target.MaxHealth = 300
				return healer, map[uint16]*entity.Player{1: healer, 2: target}
			},
			defs: []*AbilityDef{
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryDirect)},
				{Hit: HitDef{Type: HitAllyTarget}, BaseHeal: 40, Delivery: uint8(entity.DeliveryDirect)},
			},
			targetPeerID:      2,
			wantHarmonyProc:   []bool{false, false},
			wantHarmonyAmount: []float32{0, 0},
			wantTotalHeal:     []float32{46, 46}, // 40 * 1.15 each (Sympathetic Field, no Harmony)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healer, allies := tt.setup()

			for i, def := range tt.defs {
				result := resolveHeal(def, healer, allies, tt.targetPeerID)
				if result == nil {
					t.Fatalf("heal[%d]: expected non-nil HealResult", i)
				}
				if result.HarmonyProc != tt.wantHarmonyProc[i] {
					t.Errorf("heal[%d]: HarmonyProc = %v, want %v", i, result.HarmonyProc, tt.wantHarmonyProc[i])
				}
				if math.Abs(float64(result.HarmonyAmount-tt.wantHarmonyAmount[i])) > 0.01 {
					t.Errorf("heal[%d]: HarmonyAmount = %f, want %f", i, result.HarmonyAmount, tt.wantHarmonyAmount[i])
				}
				if math.Abs(float64(result.Amount-tt.wantTotalHeal[i])) > 0.01 {
					t.Errorf("heal[%d]: Amount = %f, want %f", i, result.Amount, tt.wantTotalHeal[i])
				}
			}
		})
	}
}
