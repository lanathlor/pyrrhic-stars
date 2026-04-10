package functest

import (
	"testing"
	"time"

	"codex-online/server/internal/entity"

	"github.com/google/uuid"
)

// TestVanguardDodge_StaminaDrains_InHub connects a vanguard to the hub,
// sends a dodge ability input, and verifies that the server WorldState
// reports stamina < 100 (the dodge cost of 20 was deducted).
func TestVanguardDodge_StaminaDrains_InHub(t *testing.T) {
	addr := skipIfNoGateway(t)
	charName := "Dodge_" + uuid.New().String()[:8]
	c := connectAndCreate(t, addr, "DodgeTest", "vanguard", charName)

	// Wait until we appear in the WorldState with full stamina.
	var me *PlayerState
	deadline := time.Now().Add(5 * time.Second)
	for me == nil && time.Now().Before(deadline) {
		ws := c.WaitWorldState(5 * time.Second)
		me = ws.Player(c.PeerID)
	}
	if me == nil {
		t.Fatal("local player never appeared in WorldState")
	}

	t.Logf("before dodge: stamina=%.1f class=%s", me.Stamina, me.ClassName)
	if me.Stamina != 100.0 {
		t.Fatalf("initial stamina = %.1f, want 100.0", me.Stamina)
	}

	// Send dodge ability input
	c.SendAbility(entity.ActionDodge)

	// Wait a few ticks for the server to process and send updated stamina.
	// The server processes at 20Hz, so we need at least 2-3 ticks.
	var staminaAfter float32 = 100.0
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ws := c.WaitWorldState(2 * time.Second)
		p := ws.Player(c.PeerID)
		if p != nil {
			staminaAfter = p.Stamina
			t.Logf("  tick %d: stamina=%.1f", ws.Tick, p.Stamina)
			if p.Stamina < 100.0 {
				break // stamina was deducted
			}
		}
	}

	if staminaAfter >= 100.0 {
		t.Errorf("stamina after dodge = %.1f, want < 100.0 (dodge should cost 20)", staminaAfter)
	}
	assertNear(t, staminaAfter, 80.0, 5.0, "stamina after dodge")
}
