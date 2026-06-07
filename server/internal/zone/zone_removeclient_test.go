package zone

import (
	"testing"
	"time"

	"codex-online/server/internal/level"
	"codex-online/server/internal/system"
)

// TestRemoveClient_DoesNotDeadlockUnderTick verifies that RemoveClient
// completes promptly while the zone tick loop is running.
func TestRemoveClient_DoesNotDeadlockUnderTick(t *testing.T) {
	lvl, err := level.Load("hub")
	if err != nil {
		t.Fatalf("load hub level: %v", err)
	}
	z := New("hub", lvl)

	go z.Run(t.Context())

	z.AddClient(&Client{
		PeerID:   1,
		Username: "TestPlayer",
		Send:     func([]byte) {},
	})

	time.Sleep(150 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		z.RemoveClient(1)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RemoveClient blocked for >2s — likely deadlock on z.mu")
	}

	if z.ClientCount() != 0 {
		t.Errorf("ClientCount after remove = %d, want 0", z.ClientCount())
	}
}

// TestRemoveClient_DevMode_DoesNotDeadlock is the critical regression test.
// In dev mode, botMgr is non-nil and OnRescale is wired to
// RescaleForPlayerCount which locks z.mu. If RemoveClient calls
// DismissAllForOwner (which triggers rescale) while holding z.mu,
// that's a reentrant lock — instant deadlock.
// This is the exact bug that blocked portal enter in production.
func TestRemoveClient_DevMode_DoesNotDeadlock(t *testing.T) {
	t.Setenv("CODEX_DEV", "1")

	lvl, err := level.Load("hub")
	if err != nil {
		t.Fatalf("load hub level: %v", err)
	}
	z := New("hub", lvl)

	go z.Run(t.Context())

	z.AddClient(&Client{
		PeerID:   1,
		Username: "TestPlayer",
		Send:     func([]byte) {},
	})

	time.Sleep(150 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		z.RemoveClient(1)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RemoveClient deadlocked in dev mode — reentrant z.mu via botMgr.OnRescale")
	}
}

// TestRemoveClient_WithUDPClient_DoesNotDeadlock is the same test but with
// a client that has SendUDP set, matching the production code path
// where broadcastBufUDP calls c.SendUDP during the tick.
func TestRemoveClient_WithUDPClient_DoesNotDeadlock(t *testing.T) {
	lvl, err := level.Load("hub")
	if err != nil {
		t.Fatalf("load hub level: %v", err)
	}
	z := New("hub", lvl)

	go z.Run(t.Context())

	z.AddClient(&Client{
		PeerID:   1,
		Username: "TestPlayer",
		Send:     func([]byte) {},
		SendUDP:  func([]byte) {},
	})

	time.Sleep(150 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		z.RemoveClient(1)
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("RemoveClient with UDP client blocked for >2s — likely deadlock on z.mu")
	}
}

// TestRemoveClient_WhileQueueingInput tests RemoveClient under contention
// with QueueInput (simulating the UDP read loop sending player input).
func TestRemoveClient_WhileQueueingInput(t *testing.T) {
	lvl, err := level.Load("hub")
	if err != nil {
		t.Fatalf("load hub level: %v", err)
	}
	z := New("hub", lvl)

	go z.Run(t.Context())

	z.AddClient(&system.Client{
		PeerID:   1,
		Username: "TestPlayer",
		Send:     func([]byte) {},
		SendUDP:  func([]byte) {},
	})

	time.Sleep(100 * time.Millisecond)

	// Hammer QueueInput from another goroutine (simulates UDP read loop).
	stopInput := make(chan struct{})
	go func() {
		payload := make([]byte, 25) // player input size
		for {
			select {
			case <-stopInput:
				return
			default:
				z.QueueInput(1, 0x0030, payload)
				time.Sleep(time.Millisecond) // ~1000 inputs/sec
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		z.RemoveClient(1)
		close(done)
	}()

	select {
	case <-done:
		close(stopInput)
	case <-time.After(2 * time.Second):
		close(stopInput)
		t.Fatal("RemoveClient blocked under QueueInput contention — likely deadlock on z.mu")
	}
}
