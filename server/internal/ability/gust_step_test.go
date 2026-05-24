package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestGustStepHandler(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name         string
		setup        func() *entity.Player
		wantOK       bool
		wantReason   string
		wantIFrames  bool
		wantCooldown bool
		wantGCD      bool
	}{
		{
			name: "success: spends flux and grants i-frames",
			setup: func() *entity.Player {
				return newHarmonist(1)
			},
			wantOK:       true,
			wantIFrames:  true,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "rejects on insufficient flux",
			setup: func() *entity.Player {
				p := newHarmonist(1)
				p.SetAllFluxPoolsCurrent(1)
				return p
			},
			wantOK:     false,
			wantReason: "insufficient aerokinetic flux",
		},
		{
			name: "rejects on GCD",
			setup: func() *entity.Player {
				p := newHarmonist(1)
				p.GCDTimer = 0.5
				return p
			},
			wantOK:     false,
			wantReason: "gcd",
		},
		{
			name: "rejects on cooldown",
			setup: func() *entity.Player {
				p := newHarmonist(1)
				p.Cooldowns["gust_step"] = 5.0
				return p
			},
			wantOK:     false,
			wantReason: "cooldown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setup()

			var stacksBefore int
			if p.Confluence != nil {
				stacksBefore = p.Confluence.Stacks
			}

			result := eng.Commit("gust_step", &CommitContext{
				Committer: p,
			})

			if result.OK != tt.wantOK {
				t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
			}
			if !tt.wantOK {
				if result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
				return
			}

			// Check i-frames.
			if tt.wantIFrames {
				if !p.Invincible {
					t.Error("expected Invincible = true")
				}
				if p.InvincibleTimer <= 0 {
					t.Error("expected InvincibleTimer > 0")
				}
			}

			// Check cooldown.
			if tt.wantCooldown {
				cd, ok := p.Cooldowns["gust_step"]
				if !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				} else if math.Abs(float64(cd-10.0)) > 0.1 {
					t.Errorf("cooldown = %.1f, want 10.0", cd)
				}
			}

			// Check GCD.
			if tt.wantGCD {
				if p.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
			}

			// Check confluence.
			if p.Confluence != nil && p.Confluence.Stacks != stacksBefore+1 {
				t.Errorf("Confluence stacks = %d, want %d", p.Confluence.Stacks, stacksBefore+1)
			}

			// Check aerokinetic flux was spent (secondary school: 10 * 1.25 = 12.5).
			pool := p.FluxCommit.GetPool("aerokinetic")
			if pool != nil {
				// Default aerokinetic pool: 10% of 160 = 16. After spend: 16 - 12.5 = 3.5
				expected := float32(16.0 - 12.5)
				if math.Abs(float64(pool.Current-expected)) > 0.5 {
					t.Errorf("aerokinetic flux = %.1f, want ~%.1f", pool.Current, expected)
				}
			}
		})
	}
}
