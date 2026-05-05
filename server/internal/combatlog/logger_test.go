package combatlog

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockRepo implements Repository for testing.
type mockRepo struct {
	mu        sync.Mutex
	events    []LogEntry
	instances []InstanceLog
	closed    bool
}

func (r *mockRepo) InsertEvents(_ context.Context, events []LogEntry) error {
	r.mu.Lock()
	r.events = append(r.events, events...)
	r.mu.Unlock()
	return nil
}

func (r *mockRepo) InsertInstance(_ context.Context, inst InstanceLog) error {
	r.mu.Lock()
	r.instances = append(r.instances, inst)
	r.mu.Unlock()
	return nil
}

func (r *mockRepo) InsertReplay(_ context.Context, _ string, _ [][]byte) error {
	return nil
}

func (r *mockRepo) Close() error {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	return nil
}

func (r *mockRepo) eventCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *mockRepo) instanceCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.instances)
}

func TestLogger_FlushOnClose(t *testing.T) {
	repo := &mockRepo{}
	l := NewLogger(repo, WithBatchSize(1000), WithFlushInterval(10*time.Second))

	// Log a few events
	for i := range 5 {
		l.Log(LogEntry{EventType: EventDamage, Amount: float32(i)})
	}

	// Close should flush remaining
	if err := l.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	if repo.eventCount() != 5 {
		t.Errorf("events = %d, want 5", repo.eventCount())
	}
	if !repo.closed {
		t.Error("repo should be closed")
	}
}

func TestLogger_FlushOnBatchSize(t *testing.T) {
	repo := &mockRepo{}
	l := NewLogger(repo, WithBatchSize(3), WithFlushInterval(10*time.Second))

	for i := range 3 {
		l.Log(LogEntry{EventType: EventDamage, Amount: float32(i)})
	}

	// Wait for the batch to flush
	time.Sleep(50 * time.Millisecond)

	if repo.eventCount() < 3 {
		t.Errorf("events = %d, want >= 3 (batch flushed)", repo.eventCount())
	}

	_ = l.Close()
}

func TestLogger_FlushOnInterval(t *testing.T) {
	repo := &mockRepo{}
	l := NewLogger(repo, WithBatchSize(1000), WithFlushInterval(50*time.Millisecond))

	l.Log(LogEntry{EventType: EventDamage, Amount: 42})

	// Wait for timer flush
	time.Sleep(100 * time.Millisecond)

	if repo.eventCount() < 1 {
		t.Error("expected timer flush to deliver the event")
	}

	_ = l.Close()
}

func TestLogger_NonBlockingOnFullChannel(t *testing.T) {
	repo := &mockRepo{}
	l := NewLogger(repo, WithBatchSize(1000), WithBufferSize(2), WithFlushInterval(10*time.Second))

	// Fill the channel and then some — should not block
	done := make(chan struct{})
	go func() {
		for range 100 {
			l.Log(LogEntry{EventType: EventDamage})
		}
		close(done)
	}()

	select {
	case <-done:
		// Success — Log did not block
	case <-time.After(2 * time.Second):
		t.Fatal("Log blocked on full channel")
	}

	_ = l.Close()
}

func TestLogger_InstanceEvents(t *testing.T) {
	repo := &mockRepo{}
	l := NewLogger(repo)

	l.LogInstance(InstanceLog{InstanceID: "test-inst", Outcome: OutcomePlayerWin})

	_ = l.Close()

	if repo.instanceCount() != 1 {
		t.Errorf("instances = %d, want 1", repo.instanceCount())
	}
}
