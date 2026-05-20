package entity

import (
	"math"
	"testing"
)

// --- NewPlayer ---

func TestNewPlayerGunner(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	if p.MaxHealth != 150 {
		t.Errorf("gunner max health = %f, want 150", p.MaxHealth)
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, want %f (full)", p.Health, p.MaxHealth)
	}
	if !p.Alive {
		t.Error("should be alive")
	}
}

func TestNewPlayerVanguard(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	if p.MaxHealth != 200 {
		t.Errorf("vanguard max health = %f, want 200", p.MaxHealth)
	}
	if p.GetResource("stamina") != 100 {
		t.Errorf("stamina = %f, want 100", p.GetResource("stamina"))
	}
}

func TestNewPlayerBladeDancer(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	if p.MaxHealth != 150 {
		t.Errorf("blade_dancer max health = %f, want 150", p.MaxHealth)
	}
}

func TestNewPlayerUnknownClass(t *testing.T) {
	p := NewPlayer(1, "unknown")
	if p.MaxHealth != 150 {
		t.Errorf("unknown class max health = %f, want 150 (default)", p.MaxHealth)
	}
}

// --- ApplyDamage ---

func TestApplyDamageBasic(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	dealt := p.ApplyDamage(50)
	if dealt != 50 {
		t.Errorf("dealt = %f, want 50", dealt)
	}
	if p.Health != 100 {
		t.Errorf("health = %f, want 100", p.Health)
	}
}

func TestApplyDamageKills(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	dealt := p.ApplyDamage(200)
	if dealt != 200 {
		t.Errorf("dealt = %f, want 200", dealt)
	}
	if p.Health != 0 {
		t.Errorf("health = %f, want 0", p.Health)
	}
	if p.Alive {
		t.Error("should be dead")
	}
	if p.State != PlayerStateDead {
		t.Errorf("state = %d, want PlayerStateDead", p.State)
	}
}

func TestApplyDamageToDeadPlayer(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.Alive = false
	p.State = PlayerStateDead
	dealt := p.ApplyDamage(50)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (dead player)", dealt)
	}
}

func TestApplyDamageInvincible(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.Invincible = true
	dealt := p.ApplyDamage(100)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (invincible)", dealt)
	}
	if p.Health != p.MaxHealth {
		t.Error("health should not change while invincible")
	}
}

func TestApplyDamageVanguardParry(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	// Parry: full damage reduction (Value=0.0 means 100% reduction)
	p.AddBuff(ActiveBuff{ID: "vg_parry", Type: BuffDamageReduction, Value: 0.0, Duration: 0.15})
	dealt := p.ApplyDamage(100)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (parry)", dealt)
	}
	if p.Health != p.MaxHealth {
		t.Error("health should not change during parry")
	}
}

func TestApplyDamageVanguardBlock(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	// Block: 70% damage reduction (Value=0.3 means 30% damage passes through)
	p.AddBuff(ActiveBuff{ID: "vg_block", Type: BuffDamageReduction, Value: 0.3, Duration: 1.5})
	dealt := p.ApplyDamage(100)
	// 100 * 0.3 = 30
	if dealt < 29.9 || dealt > 30.1 {
		t.Errorf("dealt = %f, want ~30.0 (70%% block)", dealt)
	}
	if p.Health < p.MaxHealth-30.1 || p.Health > p.MaxHealth-29.9 {
		t.Errorf("health = %f, want ~%f", p.Health, p.MaxHealth-30)
	}
}

func TestApplyDamageVanguardBladeSwirl(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	// Blade swirl: 20% damage reduction (Value=0.8 means 80% damage passes through)
	p.AddBuff(ActiveBuff{ID: "vortex", Type: BuffDamageReduction, Value: 0.8, Duration: 1.5})
	dealt := p.ApplyDamage(100)
	expected := float32(80.0) // 100 * 0.8
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (20%% DR from swirl)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerGuard(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	// Guard: 50% damage reduction (Value=0.5)
	p.AddBuff(ActiveBuff{ID: "guard", Type: BuffDamageReduction, Value: 0.5, Duration: 1.5})
	dealt := p.ApplyDamage(100)
	expected := float32(50.0) // 100 * 0.5
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (50%% guard)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerDR(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	// BD DR buff: 30% damage reduction (Value=0.7 means 70% damage passes through)
	p.AddBuff(ActiveBuff{ID: "bd_dr", Type: BuffDamageReduction, Value: 0.7, Duration: 3.0})
	dealt := p.ApplyDamage(100)
	expected := float32(70.0)
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (DR factor 0.7)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerShieldAbsorb(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.Resources["shield"].Current = 20.0
	dealt := p.ApplyDamage(50)
	// Shield absorbs 20, remaining 30 goes to health
	if dealt != 30 {
		t.Errorf("dealt = %f, want 30 (50 - 20 shield)", dealt)
	}
	if p.Resources["shield"].Current != 0 {
		t.Errorf("shield = %f, want 0", p.Resources["shield"].Current)
	}
	if p.Health != p.MaxHealth-30 {
		t.Errorf("health = %f, want %f", p.Health, p.MaxHealth-30)
	}
}

func TestApplyDamageBladeDancerShieldFullAbsorb(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.Resources["shield"].Current = 25.0
	dealt := p.ApplyDamage(20)
	if dealt != 20 {
		t.Errorf("dealt = %f, want 20", dealt)
	}
	if p.Resources["shield"].Current != 5 {
		t.Errorf("shield = %f, want 5 (25-20)", p.Resources["shield"].Current)
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, should be untouched", p.Health)
	}
}

func TestApplyDamageVanguardBlockPlusSwirl(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	// Both block (0.3) and swirl (0.8) stack multiplicatively
	p.AddBuff(ActiveBuff{ID: "vg_block", Type: BuffDamageReduction, Value: 0.3, Duration: 1.5})
	p.AddBuff(ActiveBuff{ID: "vortex", Type: BuffDamageReduction, Value: 0.8, Duration: 1.5})
	dealt := p.ApplyDamage(100)
	expected := float32(100 * 0.3 * 0.8) // 24
	if dealt < expected-0.1 || dealt > expected+0.1 {
		t.Errorf("dealt = %f, want ~%f (block+swirl stacked)", dealt, expected)
	}
}

// --- Forward / AimDirection ---

func TestForwardAtZeroYaw(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.RotationY = 0
	f := p.Forward()
	// At rotY=0, forward = (0, 0, -1)
	if f.Z > -0.99 {
		t.Errorf("forward = %v, want (0,0,-1)", f)
	}
}

func TestAimDirectionFlat(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.RotationY = 0
	p.AimPitch = 0
	d := p.AimDirection()
	if d.Z > -0.99 || d.Y > 0.01 {
		t.Errorf("aim dir = %v, want (0,0,-1)", d)
	}
}

func TestAimDirectionWithPitch(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.RotationY = 0
	p.AimPitch = float32(math.Pi / 4) // 45 degrees up
	d := p.AimDirection()
	if d.Y < 0.5 {
		t.Errorf("aim Y = %f, should be positive with upward pitch", d.Y)
	}
}

// --- EyePosition ---

func TestEyePosition(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.Position = Vec3{X: 1, Y: 0, Z: 2}
	eye := p.EyePosition()
	if eye.Y != 1.6 {
		t.Errorf("eye Y = %f, want 1.6", eye.Y)
	}
	if eye.X != 1 || eye.Z != 2 {
		t.Error("eye XZ should match position")
	}
}

// --- Movement() ---

func TestMovementGunner(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	m := p.Movement()
	if m.WalkSpeed != 5.5 {
		t.Errorf("gunner walk speed = %f, want 5.5", m.WalkSpeed)
	}
}

func TestMovementVanguard(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	m := p.Movement()
	if m.WalkSpeed != 5.0 {
		t.Errorf("vanguard walk speed = %f, want 5.0", m.WalkSpeed)
	}
}

func TestMovementBladeDancer(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	m := p.Movement()
	if m.WalkSpeed != 6.0 {
		t.Errorf("blade_dancer walk speed = %f, want 6.0", m.WalkSpeed)
	}
}

func TestMovementUnknownFallsBackToGunner(t *testing.T) {
	p := NewPlayer(1, "unknown")
	m := p.Movement()
	gunner := Classes[ClassGunner].Movement
	if m != gunner {
		t.Errorf("unknown class movement = %+v, want gunner defaults", m)
	}
}

// --- ApplyDamage Plating floor ---

func TestApplyDamage_PlatingFloor(t *testing.T) {
	tests := []struct {
		name      string
		plating   float32
		damage    float32
		wantDealt float32
	}{
		{"no plating", 0, 50, 50},
		{"plating partial, above floor", 10, 50, 40},
		{"plating exceeds 80%, clamped to floor", 45, 50, 10},
		{"plating exceeds damage, clamped to floor", 100, 50, 10},
		{"small hit, above floor", 5, 10, 5},
		{"small hit, below floor, clamped", 9, 10, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, ClassGunner)
			p.GearStats.Plating = tc.plating
			startHP := p.Health
			dealt := p.ApplyDamage(tc.damage)
			if dealt != tc.wantDealt {
				t.Errorf("dealt = %f, want %f", dealt, tc.wantDealt)
			}
			wantHP := startHP - tc.wantDealt
			if p.Health != wantHP {
				t.Errorf("health = %f, want %f", p.Health, wantHP)
			}
		})
	}
}

// --- TempoMult ---

func TestTempoMult(t *testing.T) {
	tests := []struct {
		name  string
		tempo float32
		want  float32
	}{
		{"zero tempo", 0, 1.0},
		{"50 tempo", 50, 1.5},
		{"100 tempo", 100, 2.0},
		{"12.5 tempo", 12.5, 1.125},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, ClassGunner)
			p.GearStats.Tempo = tc.tempo
			got := p.TempoMult()
			if got != tc.want {
				t.Errorf("TempoMult() = %f, want %f", got, tc.want)
			}
		})
	}
}
