package functest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubPersistence_PositionSavedOnDisconnect(t *testing.T) {
	addr := skipIfNoGateway(t)
	playerUUID := uuid.New().String()
	charName := "Persist_" + uuid.New().String()[:8]

	// Session 1: create character, walk, disconnect.
	c1 := DialWithUUID(t, addr, playerUUID, "PersistTest")
	c1.WaitCharacterList(5 * time.Second)
	cs1 := c1.CreateCharacter("gunner", charName)
	charID := cs1.CharID

	var me *PlayerState
	deadline := time.Now().Add(5 * time.Second)
	for me == nil && time.Now().Before(deadline) {
		ws := c1.WaitWorldState(5 * time.Second)
		me = ws.Player(c1.PeerID)
	}
	if me == nil {
		t.Fatal("local player never appeared in WorldState")
	}

	// Walk in small steps.
	walkX := float32(20.0)
	walkY := float32(-199.9)
	walkZ := float32(-88.0)
	walkRotY := float32(1.0)

	startX := me.PosX
	startZ := me.PosZ
	steps := 10
	for tick := uint32(1); tick <= uint32(steps); tick++ {
		frac := float32(tick) / float32(steps)
		x := startX + (walkX-startX)*frac
		z := startZ + (walkZ-startZ)*frac
		c1.SendPlayerInput(x, walkY, z, walkRotY, tick)
		time.Sleep(60 * time.Millisecond)
	}

	var confirmed bool
	deadline = time.Now().Add(3 * time.Second)
	for !confirmed && time.Now().Before(deadline) {
		ws := c1.WaitWorldState(3 * time.Second)
		p := ws.Player(c1.PeerID)
		if p != nil && p.PosX > 18.0 && p.PosZ < -85.0 {
			confirmed = true
		}
	}
	if !confirmed {
		t.Fatal("server never accepted walked position")
	}

	c1.Close()
	time.Sleep(300 * time.Millisecond)

	// Session 2: reconnect, select same character by ID, verify position.
	c2 := DialWithUUID(t, addr, playerUUID, "PersistTest")
	cl := c2.WaitCharacterList(5 * time.Second)
	if len(cl.Characters) == 0 {
		t.Fatal("expected at least 1 character")
	}

	cs2 := c2.SelectCharacter(charID)
	t.Logf("restored: pos=(%.1f, %.1f, %.1f) rotY=%.4f", cs2.PosX, cs2.PosY, cs2.PosZ, cs2.RotY)

	assertNear(t, cs2.PosX, walkX, 2.0, "restored X")
	assertNear(t, cs2.PosY, walkY, 2.0, "restored Y")
	assertNear(t, cs2.PosZ, walkZ, 2.0, "restored Z")
	assertNear(t, cs2.RotY, walkRotY, 0.2, "restored RotY")
}
