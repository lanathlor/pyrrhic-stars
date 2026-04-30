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
	if p.Stamina != 100 {
		t.Errorf("stamina = %f, want 100", p.Stamina)
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
	p.IsBlocking = true
	p.ParryTimer = 0.2 // active parry window
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
	p.IsBlocking = true
	p.ParryTimer = 0 // not in parry window
	dealt := p.ApplyDamage(100)
	// 100 * 0.3 = 30 (70% block)
	if dealt < 29.9 || dealt > 30.1 {
		t.Errorf("dealt = %f, want ~30.0 (70%% block)", dealt)
	}
	if p.Health < p.MaxHealth-30.1 || p.Health > p.MaxHealth-29.9 {
		t.Errorf("health = %f, want ~%f", p.Health, p.MaxHealth-30)
	}
}

func TestApplyDamageVanguardBladeSwirl(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	p.BladeSwirl = true
	dealt := p.ApplyDamage(100)
	expected := float32(80.0) // 100 * 0.8
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (20%% DR from swirl)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerGuard(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.GuardActive = true
	dealt := p.ApplyDamage(100)
	expected := float32(50.0) // 100 * 0.5
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (50%% guard)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerDR(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.BDDRFactor = 0.7 // 30% damage reduction
	p.BDDRTimer = 3.0
	dealt := p.ApplyDamage(100)
	expected := float32(70.0)
	if dealt != expected {
		t.Errorf("dealt = %f, want %f (DR factor 0.7)", dealt, expected)
	}
}

func TestApplyDamageBladeDancerShieldAbsorb(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.BDShieldHP = 20.0
	dealt := p.ApplyDamage(50)
	// ApplyDamage returns the amount parameter after modifications, shield absorbs 20, 30 goes to HP
	// The function returns `amount` which is the post-shield remainder = 30, plus the absorbed part is returned via the earlier shield branch
	// Actually: shield absorbs 20 → amount becomes 30 → health -= 30 → return 30
	// Wait, let me re-read the code...
	// When amount > shield: amount -= shield, shield = 0, then health -= amount, return amount
	// So dealt = 30 (only the health damage portion)
	// No: the function has TWO return paths for shield:
	//   1. amount <= shield → return amount (full absorbed)
	//   2. amount > shield → amount -= shield, shield = 0 → fall through to health -= amount → return amount (the remainder)
	if dealt != 30 {
		t.Errorf("dealt = %f, want 30 (50 - 20 shield)", dealt)
	}
	if p.BDShieldHP != 0 {
		t.Errorf("shield = %f, want 0", p.BDShieldHP)
	}
	if p.Health != p.MaxHealth-30 {
		t.Errorf("health = %f, want %f", p.Health, p.MaxHealth-30)
	}
}

func TestApplyDamageBladeDancerShieldFullAbsorb(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	p.BDShieldHP = 25.0
	dealt := p.ApplyDamage(20)
	if dealt != 20 {
		t.Errorf("dealt = %f, want 20", dealt)
	}
	if p.BDShieldHP != 5 {
		t.Errorf("shield = %f, want 5 (25-20)", p.BDShieldHP)
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, should be untouched", p.Health)
	}
}

func TestApplyDamageVanguardBlockPlusSwirl(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	p.IsBlocking = true
	p.BladeSwirl = true
	// Both block (0.3) and swirl (0.8) stack multiplicatively
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
	p.AimPitch = float32(math.Pi / 4) // 45° up
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

// --- stats() ---

func TestStatsGunner(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	s := p.stats()
	if s.WalkSpeed != 5.5 {
		t.Errorf("gunner walk speed = %f, want 5.5", s.WalkSpeed)
	}
}

func TestStatsVanguard(t *testing.T) {
	p := NewPlayer(1, ClassVanguard)
	s := p.stats()
	if s.WalkSpeed != 5.0 {
		t.Errorf("vanguard walk speed = %f, want 5.0", s.WalkSpeed)
	}
}

func TestStatsBladeDancer(t *testing.T) {
	p := NewPlayer(1, ClassBladeDancer)
	s := p.stats()
	if s.WalkSpeed != 6.0 {
		t.Errorf("blade_dancer walk speed = %f, want 6.0", s.WalkSpeed)
	}
}

func TestStatsUnknownFallsBackToGunner(t *testing.T) {
	p := NewPlayer(1, "unknown")
	s := p.stats()
	gunner := classStatsTable[ClassGunner]
	if s != gunner {
		t.Errorf("unknown class stats = %+v, want gunner defaults", s)
	}
}

// --- max32 ---

func TestMax32(t *testing.T) {
	if got := max32(3, 5); got != 5 {
		t.Errorf("max32(3,5) = %f, want 5", got)
	}
	if got := max32(7, 2); got != 7 {
		t.Errorf("max32(7,2) = %f, want 7", got)
	}
	if got := max32(4, 4); got != 4 {
		t.Errorf("max32(4,4) = %f, want 4", got)
	}
}
