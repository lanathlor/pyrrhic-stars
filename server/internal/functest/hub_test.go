package functest

import (
	"math"
	"os"
	"testing"
	"time"

	"codex-online/server/internal/entity"

	"github.com/google/uuid"
)

func gatewayAddr() string {
	if addr := os.Getenv("GATEWAY_ADDR"); addr != "" {
		return addr
	}
	return DefaultAddr
}

func skipIfNoGateway(t *testing.T) string {
	t.Helper()
	addr := gatewayAddr()
	c, err := TryDial(addr, "Probe")
	if err != nil {
		t.Skipf("skipping: gateway not reachable at %s: %v", addr, err)
	}
	c.Close()
	return addr
}

// connectAndCreate is the standard new-character flow: dial, wait for char list, create.
func connectAndCreate(t *testing.T, addr, username, class, charName string) *Client {
	t.Helper()
	c := Dial(t, addr, username)
	c.WaitCharacterList(5 * time.Second)
	c.CreateCharacter(class, charName)
	return c
}

func assertNear(t *testing.T, got, want, tolerance float32, label string) {
	t.Helper()
	if diff := float32(math.Abs(float64(got - want))); diff > tolerance {
		t.Errorf("%s = %.2f, want %.2f (±%.2f, diff=%.2f)", label, got, want, tolerance, diff)
	}
}

// ---------------------------------------------------------------------------
// Hub spawn tests
// ---------------------------------------------------------------------------

func TestHubSpawn_Position(t *testing.T) {
	addr := skipIfNoGateway(t)
	name := "Spawn_" + uuid.New().String()[:8]
	c := connectAndCreate(t, addr, "SpawnTest", entity.ClassGunner, name)

	var me *PlayerState
	deadline := time.Now().Add(5 * time.Second)
	for me == nil && time.Now().Before(deadline) {
		ws := c.WaitWorldState(5 * time.Second)
		me = ws.Player(c.PeerID)
	}
	if me == nil {
		t.Fatal("local player never appeared in WorldState")
	}

	t.Logf("spawn: pos=(%.1f, %.1f, %.1f) rotY=%.4f class=%s name=%s",
		me.PosX, me.PosY, me.PosZ, me.RotY, me.ClassName, me.Username)

	assertNear(t, me.PosY, 100.15, 1.0, "spawn Y (upper plaza)")
	if me.PosX < 31.0 || me.PosX > 35.0 {
		t.Errorf("spawn X = %.1f, want 31-35", me.PosX)
	}
	if me.PosZ < 2.0 || me.PosZ > 6.0 {
		t.Errorf("spawn Z = %.1f, want 2-6", me.PosZ)
	}
}

func TestHubSpawn_FacingDirection(t *testing.T) {
	addr := skipIfNoGateway(t)
	name := "Yaw_" + uuid.New().String()[:8]
	c := connectAndCreate(t, addr, "YawTest", entity.ClassGunner, name)

	var me *PlayerState
	deadline := time.Now().Add(5 * time.Second)
	for me == nil && time.Now().Before(deadline) {
		ws := c.WaitWorldState(5 * time.Second)
		me = ws.Player(c.PeerID)
	}
	if me == nil {
		t.Fatal("local player never appeared in WorldState")
	}

	wantYaw := float32(math.Pi)
	assertNear(t, me.RotY, wantYaw, 0.1, "spawn yaw (facing south)")
}

func TestHubSpawn_NPCsPresent(t *testing.T) {
	addr := skipIfNoGateway(t)
	name := "NPC_" + uuid.New().String()[:8]
	c := connectAndCreate(t, addr, "NPCTest", entity.ClassGunner, name)

	var ws *WorldState
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ws = c.WaitWorldState(5 * time.Second)
		if len(ws.NPCs) > 0 {
			break
		}
	}
	if len(ws.NPCs) == 0 {
		t.Fatal("no NPCs in hub WorldState")
	}
	t.Logf("hub has %d NPCs", len(ws.NPCs))
}

func TestHubSpawn_CharacterNameOverhead(t *testing.T) {
	addr := skipIfNoGateway(t)
	charName := "Hero_" + uuid.New().String()[:8]
	c := connectAndCreate(t, addr, "NameTest", entity.ClassGunner, charName)

	var me *PlayerState
	deadline := time.Now().Add(5 * time.Second)
	for me == nil && time.Now().Before(deadline) {
		ws := c.WaitWorldState(5 * time.Second)
		me = ws.Player(c.PeerID)
	}
	if me == nil {
		t.Fatal("local player never appeared in WorldState")
	}

	// WorldState Username field should contain the character name, not account name.
	if me.Username != charName {
		t.Errorf("overhead name = %q, want %q (character name)", me.Username, charName)
	}
}

// ---------------------------------------------------------------------------
// Character creation tests
// ---------------------------------------------------------------------------

func TestCreateCharacter_Success(t *testing.T) {
	addr := skipIfNoGateway(t)
	charName := "New_" + uuid.New().String()[:8]

	c := Dial(t, addr, "CreateTest")
	cl := c.WaitCharacterList(5 * time.Second)
	t.Logf("initial: %d chars, username=%q", len(cl.Characters), cl.Username)

	cs := c.CreateCharacter(entity.ClassGunner, charName)
	t.Logf("created: id=%d class=%s name=%s", cs.CharID, cs.ClassName, cs.Name)

	if cs.CharID == 0 {
		t.Error("CharID should be > 0")
	}
	if cs.ClassName != entity.ClassGunner {
		t.Errorf("class = %q, want gunner", cs.ClassName)
	}
	if cs.Name != charName {
		t.Errorf("name = %q, want %q", cs.Name, charName)
	}
}

func TestCreateCharacter_DuplicateName(t *testing.T) {
	addr := skipIfNoGateway(t)
	charName := "Dup_" + uuid.New().String()[:8]

	// Player 1 creates the character.
	c1 := connectAndCreate(t, addr, "Dup1", entity.ClassGunner, charName)
	_ = c1

	// Player 2 tries the same name.
	c2 := Dial(t, addr, "Dup2")
	c2.WaitCharacterList(5 * time.Second)
	c2.SendCreateCharacter(entity.ClassVanguard, charName)

	err := c2.WaitCharacterError(5 * time.Second)
	t.Logf("error: code=%d msg=%q", err.Code, err.Message)

	if err.Code != 1 {
		t.Errorf("error code = %d, want 1 (name taken)", err.Code)
	}
}

func TestCreateCharacter_InvalidName(t *testing.T) {
	addr := skipIfNoGateway(t)

	c := Dial(t, addr, "InvalidTest")
	c.WaitCharacterList(5 * time.Second)

	// Too short.
	c.SendCreateCharacter(entity.ClassGunner, "A")
	err := c.WaitCharacterError(5 * time.Second)
	if err.Code != 3 {
		t.Errorf("too-short: code=%d, want 3", err.Code)
	}
}

func TestCreateCharacter_MultiplePerClass(t *testing.T) {
	addr := skipIfNoGateway(t)
	playerUUID := uuid.New().String()

	// Create 3 gunners.
	for i := range 3 {
		name := "Gun_" + uuid.New().String()[:8]
		c := DialWithUUID(t, addr, playerUUID, "MultiClass")
		c.WaitCharacterList(5 * time.Second)
		cs := c.CreateCharacter(entity.ClassGunner, name)
		t.Logf("created gunner #%d: id=%d name=%s", i+1, cs.CharID, cs.Name)
		c.Close()
		time.Sleep(200 * time.Millisecond)
	}

	// Reconnect and verify all 3 appear in the list.
	c := DialWithUUID(t, addr, playerUUID, "MultiClass")
	cl := c.WaitCharacterList(5 * time.Second)
	t.Logf("character list: %d chars", len(cl.Characters))

	gunnerCount := 0
	for _, ch := range cl.Characters {
		if ch.ClassName == entity.ClassGunner {
			gunnerCount++
		}
	}
	if gunnerCount != 3 {
		t.Errorf("gunner count = %d, want 3", gunnerCount)
	}
}

// ---------------------------------------------------------------------------
// Character selection tests
// ---------------------------------------------------------------------------

func TestSelectCharacter_ByID(t *testing.T) {
	addr := skipIfNoGateway(t)
	playerUUID := uuid.New().String()
	charName := "Sel_" + uuid.New().String()[:8]

	// Session 1: create character.
	c1 := DialWithUUID(t, addr, playerUUID, "SelTest")
	c1.WaitCharacterList(5 * time.Second)
	cs1 := c1.CreateCharacter(entity.ClassVanguard, charName)
	t.Logf("created: id=%d", cs1.CharID)
	c1.Close()
	time.Sleep(200 * time.Millisecond)

	// Session 2: select by ID.
	c2 := DialWithUUID(t, addr, playerUUID, "SelTest")
	cl := c2.WaitCharacterList(5 * time.Second)

	var charID uint32
	for _, ch := range cl.Characters {
		if ch.Name == charName {
			charID = ch.CharID
		}
	}
	if charID == 0 {
		t.Fatal("character not found in list")
	}

	cs2 := c2.SelectCharacter(charID)
	t.Logf("selected: id=%d class=%s name=%s", cs2.CharID, cs2.ClassName, cs2.Name)

	if cs2.CharID != charID {
		t.Errorf("charID = %d, want %d", cs2.CharID, charID)
	}
	if cs2.ClassName != entity.ClassVanguard {
		t.Errorf("class = %q, want vanguard", cs2.ClassName)
	}
	if cs2.Name != charName {
		t.Errorf("name = %q, want %q", cs2.Name, charName)
	}
}

// ---------------------------------------------------------------------------
// Username lock test
// ---------------------------------------------------------------------------

func TestUsername_LockedAfterFirstLogin(t *testing.T) {
	addr := skipIfNoGateway(t)
	playerUUID := uuid.New().String()

	// Session 1: connect with "OriginalName".
	c1 := DialWithUUID(t, addr, playerUUID, "OriginalName")
	cl1 := c1.WaitCharacterList(5 * time.Second)
	t.Logf("session 1 username: %q", cl1.Username)
	if cl1.Username != "OriginalName" {
		t.Errorf("username = %q, want OriginalName", cl1.Username)
	}
	c1.Close()
	time.Sleep(200 * time.Millisecond)

	// Session 2: reconnect with "DifferentName" — server should return "OriginalName".
	c2 := DialWithUUID(t, addr, playerUUID, "DifferentName")
	cl2 := c2.WaitCharacterList(5 * time.Second)
	t.Logf("session 2 username: %q", cl2.Username)
	if cl2.Username != "OriginalName" {
		t.Errorf("username = %q, want OriginalName (locked)", cl2.Username)
	}
}
