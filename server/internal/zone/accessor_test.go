package zone

import (
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/system"
	"testing"
)

// minimalLevel returns a level with minimal bounds for testing accessors.
func minimalLevel() *level.Level {
	return &level.Level{
		PlayerBoundsMinX: -10, PlayerBoundsMaxX: 10,
		PlayerBoundsMinZ: -10, PlayerBoundsMaxZ: 10,
		PlayerSpawns: []level.PlayerSpawn{{Position: entity.Vec3{X: 0, Y: 0.1, Z: 0}}},
	}
}

// fakeClient creates a system.Client that discards all messages.
func fakeClient(peerID uint16, username string) *system.Client {
	return &system.Client{
		PeerID:   peerID,
		Username: username,
		Send:     func([]byte) {},
	}
}

func TestRemoveClient(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	z.AddClient(fakeClient(1, "Alice"))

	if z.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", z.ClientCount())
	}
	z.RemoveClient(1)
	if z.ClientCount() != 0 {
		t.Errorf("expected 0 clients after remove, got %d", z.ClientCount())
	}
	if z.GetPlayer(1) != nil {
		t.Error("player should be removed along with client")
	}
}

func TestClientCount(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	if z.ClientCount() != 0 {
		t.Errorf("empty zone client count = %d, want 0", z.ClientCount())
	}
	z.AddClient(fakeClient(1, "Alice"))
	z.AddClient(fakeClient(2, "Bob"))
	if z.ClientCount() != 2 {
		t.Errorf("client count = %d, want 2", z.ClientCount())
	}
}

func TestGetPeerIDs(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	z.AddClient(fakeClient(1, "Alice"))
	z.AddClient(fakeClient(2, "Bob"))

	ids := z.GetPeerIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 peer IDs, got %d", len(ids))
	}
	has1, has2 := false, false
	for _, id := range ids {
		if id == 1 {
			has1 = true
		}
		if id == 2 {
			has2 = true
		}
	}
	if !has1 || !has2 {
		t.Errorf("peer IDs = %v, want [1, 2]", ids)
	}
}

func TestGetPlayer(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	z.AddClient(fakeClient(1, "Alice"))

	p := z.GetPlayer(1)
	if p == nil {
		t.Fatal("GetPlayer(1) returned nil")
	}
	if p.ID != 1 {
		t.Errorf("player peer ID = %d, want 1", p.ID)
	}
	if z.GetPlayer(99) != nil {
		t.Error("GetPlayer(99) should return nil")
	}
}

func TestSetPlayerPosition(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	z.AddClient(fakeClient(1, "Alice"))

	newPos := entity.Vec3{X: 5, Y: 1, Z: -3}
	z.SetPlayerPosition(1, newPos, 1.5)

	p := z.GetPlayer(1)
	if p.Position != newPos {
		t.Errorf("position = %v, want %v", p.Position, newPos)
	}
	if p.RotationY != 1.5 {
		t.Errorf("rotY = %f, want 1.5", p.RotationY)
	}
}

func TestSetPlayerPositionNonexistent(_ *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())
	// Should not panic for missing player
	z.SetPlayerPosition(99, entity.Vec3{}, 0)
}

func TestBroadcast(t *testing.T) {
	z := New("test_hub", ZoneTypeOpenWorld, minimalLevel())

	var received1, received2 int
	c1 := &system.Client{PeerID: 1, Username: "Alice", Send: func([]byte) { received1++ }}
	c2 := &system.Client{PeerID: 2, Username: "Bob", Send: func([]byte) { received2++ }}
	z.AddClient(c1)
	z.AddClient(c2)

	// Broadcast to all (exclude 0)
	z.Broadcast([]byte{0x01}, 0)
	if received1 != 1 || received2 != 1 {
		t.Errorf("broadcast all: received = (%d, %d), want (1, 1)", received1, received2)
	}

	// Broadcast excluding peer 1
	z.Broadcast([]byte{0x02}, 1)
	if received1 != 1 {
		t.Errorf("excluded peer should not receive: received1 = %d, want 1", received1)
	}
	if received2 != 2 {
		t.Errorf("non-excluded peer should receive: received2 = %d, want 2", received2)
	}
}
