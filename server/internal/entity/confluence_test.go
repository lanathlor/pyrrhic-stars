package entity

import (
	"math"
	"testing"
)

func TestConfluenceOnAbilityComplete(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		casts      int
		wantStacks int
	}{
		{"0 to 1", 0, 1, 1},
		{"0 to 3", 0, 3, 3},
		{"0 to 5 (max)", 0, 5, 5},
		{"0 to 5 capped at max", 0, 7, 5},
		{"3 to 5 capped", 3, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfluenceState{Stacks: tt.initial, MaxStacks: 5, DecayRate: 1.0}
			for i := 0; i < tt.casts; i++ {
				c.OnAbilityComplete()
			}
			if c.Stacks != tt.wantStacks {
				t.Errorf("Stacks = %d, want %d", c.Stacks, tt.wantStacks)
			}
		})
	}
}

func TestConfluenceAbilityPowerMult(t *testing.T) {
	tests := []struct {
		name     string
		stacks   int
		wantMult float32
	}{
		{"0 stacks", 0, 1.0},
		{"1 stack", 1, 1.08},
		{"2 stacks", 2, 1.16},
		{"3 stacks", 3, 1.24},
		{"4 stacks", 4, 1.32},
		{"5 stacks", 5, 1.40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfluenceState{Stacks: tt.stacks, MaxStacks: 5}
			got := c.AbilityPowerMult()
			if math.Abs(float64(got-tt.wantMult)) > 0.001 {
				t.Errorf("AbilityPowerMult() = %f, want %f", got, tt.wantMult)
			}
		})
	}
}

func TestConfluenceOnInterrupt(t *testing.T) {
	c := &ConfluenceState{Stacks: 4, MaxStacks: 5, IdleTimer: 2.0, DecayTimer: 0.5}
	c.OnInterrupt()
	if c.Stacks != 0 {
		t.Errorf("Stacks = %d, want 0", c.Stacks)
	}
	if c.IdleTimer != 0 {
		t.Errorf("IdleTimer = %f, want 0", c.IdleTimer)
	}
	if c.DecayTimer != 0 {
		t.Errorf("DecayTimer = %f, want 0", c.DecayTimer)
	}
}

func TestConfluenceTickNoDecayBeforeIdle(t *testing.T) {
	tests := []struct {
		name       string
		dt         float32
		ticks      int
		wantStacks int
	}{
		{"1s idle", 1.0, 1, 3},
		{"3.9s idle", 0.1, 39, 3},
		{"exactly 4s idle no decay yet on that tick", 4.0, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfluenceState{Stacks: 3, MaxStacks: 5, DecayRate: 1.0}
			for i := 0; i < tt.ticks; i++ {
				c.Tick(tt.dt)
			}
			if c.Stacks != tt.wantStacks {
				t.Errorf("Stacks = %d, want %d", c.Stacks, tt.wantStacks)
			}
		})
	}
}

func TestConfluenceTickDecayAfterIdle(t *testing.T) {
	tests := []struct {
		name       string
		initial    int
		idleTime   float32
		decayTime  float32
		wantStacks int
	}{
		{"4s idle then 1s decay loses 1 stack", 5, 4.0, 1.0, 4},
		{"4s idle then 3s decay loses 3 stacks", 5, 4.0, 3.0, 2},
		{"4s idle then 5s decay loses all 5", 5, 4.0, 5.0, 0},
		{"4s idle then 10s decay floors at 0", 3, 4.0, 10.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfluenceState{Stacks: tt.initial, MaxStacks: 5, DecayRate: 1.0}
			// Use 0.25s ticks to avoid float32 precision issues with 0.05.
			idleSteps := int(tt.idleTime / 0.25)
			for range idleSteps {
				c.Tick(0.25)
			}
			// Now advance decay time.
			decaySteps := int(tt.decayTime / 0.25)
			for range decaySteps {
				c.Tick(0.25)
			}
			if c.Stacks != tt.wantStacks {
				t.Errorf("Stacks = %d, want %d", c.Stacks, tt.wantStacks)
			}
		})
	}
}

func TestConfluenceTickZeroStacksNoop(t *testing.T) {
	c := &ConfluenceState{Stacks: 0, MaxStacks: 5, DecayRate: 1.0}
	c.Tick(10.0)
	if c.Stacks != 0 {
		t.Errorf("Stacks = %d, want 0", c.Stacks)
	}
	if c.IdleTimer != 0 {
		t.Errorf("IdleTimer = %f, want 0", c.IdleTimer)
	}
}

func TestConfluenceOnAbilityCompleteResetsIdleTimer(t *testing.T) {
	c := &ConfluenceState{Stacks: 2, MaxStacks: 5, DecayRate: 1.0}
	// Advance idle timer close to decay threshold.
	c.Tick(3.5)
	if c.IdleTimer < 3.0 {
		t.Fatalf("IdleTimer should be ~3.5, got %f", c.IdleTimer)
	}
	// Complete an ability, which should reset idle timer.
	c.OnAbilityComplete()
	if c.IdleTimer != 0 {
		t.Errorf("IdleTimer after ability = %f, want 0", c.IdleTimer)
	}
	if c.Stacks != 3 {
		t.Errorf("Stacks = %d, want 3", c.Stacks)
	}
	// Verify decay does not start for another 4s.
	c.Tick(3.9)
	if c.Stacks != 3 {
		t.Errorf("Stacks after 3.9s = %d, want 3 (no decay yet)", c.Stacks)
	}
}

func TestNewPlayerArcanotechnicienHasConfluence(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{SpecHarmonist, SpecHarmonist},
		{SpecDestroyer, SpecDestroyer},
		{SpecBattlemage, SpecBattlemage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlayerWithSpec(1, ClassArcanotechnicien, tt.spec)
			if p.Confluence == nil {
				t.Fatal("Confluence should be initialized for all Arcanotechnicien specs")
			}
			if p.Confluence.MaxStacks != 5 {
				t.Errorf("MaxStacks = %d, want 5", p.Confluence.MaxStacks)
			}
			if p.Confluence.Stacks != 0 {
				t.Errorf("initial Stacks = %d, want 0", p.Confluence.Stacks)
			}
		})
	}
}

func TestNewPlayerNonArcanotechnicienNoConfluence(t *testing.T) {
	classes := []string{ClassGunner, ClassVanguard, ClassBladeDancer}
	for _, cls := range classes {
		t.Run(cls, func(t *testing.T) {
			p := NewPlayer(1, cls)
			if p.Confluence != nil {
				t.Errorf("%s should not have Confluence", cls)
			}
		})
	}
}
