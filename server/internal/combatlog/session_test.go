package combatlog

import (
	"testing"
	"time"
)

func TestSession_LogEvent_AutoFills(t *testing.T) {
	sink := NewInMemorySink()
	s := NewSession(sink, "inst-1", "group-1", "arena_boss", "arena", "run-1", 0, SourceLive, 100)

	s.LogEvent(110, 0.75, 1, LogEntry{
		EventType:    EventDamage,
		SourceEntity: testPlayerEntity,
		Target:       "enemy_1000",
		Amount:       50,
	})

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.GroupID != "group-1" {
		t.Errorf("GroupID = %s, want group-1", e.GroupID)
	}
	if e.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %s, want inst-1", e.InstanceID)
	}
	if e.EncounterID != "arena_boss" {
		t.Errorf("EncounterID = %s, want arena_boss", e.EncounterID)
	}
	if e.Tick != 110 {
		t.Errorf("Tick = %d, want 110", e.Tick)
	}
	// (110-100) * 50ms = 500ms
	wantTS := 500 * time.Millisecond
	if e.Timestamp != wantTS {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, wantTS)
	}
	if e.BossHealth != 0.75 {
		t.Errorf("BossHealth = %f, want 0.75", e.BossHealth)
	}
	if e.Phase != "phase_1" {
		t.Errorf("Phase = %s, want phase_1", e.Phase)
	}
}

func TestSession_CheckPhaseChange(t *testing.T) {
	sink := NewInMemorySink()
	s := NewSession(sink, "inst-1", "group-1", "arena_boss", "arena", "run-1", 0, SourceLive, 100)

	// First call initializes — no event
	s.CheckPhaseChange(101, 1000, 1, 0.8)
	if len(sink.EventsOfType(EventPhaseChange)) != 0 {
		t.Error("first CheckPhaseChange should not emit (initial)")
	}

	// Same phase — no event
	s.CheckPhaseChange(102, 1000, 1, 0.7)
	if len(sink.EventsOfType(EventPhaseChange)) != 0 {
		t.Error("same phase should not emit")
	}

	// Phase change — event emitted
	s.CheckPhaseChange(103, 1000, 2, 0.55)
	changes := sink.EventsOfType(EventPhaseChange)
	if len(changes) != 1 {
		t.Fatalf("phase change events = %d, want 1", len(changes))
	}
	if changes[0].Target != "enemy_1000" {
		t.Errorf("Target = %s, want enemy_1000", changes[0].Target)
	}

	// Another phase change
	s.CheckPhaseChange(200, 1000, 3, 0.25)
	if len(sink.EventsOfType(EventPhaseChange)) != 2 {
		t.Error("should have 2 phase changes total")
	}
}

func TestSession_AddParticipant(t *testing.T) {
	sink := NewInMemorySink()
	s := NewSession(sink, "inst-1", "group-1", "arena_boss", "arena", "run-1", 0, SourceLive, 0)

	s.AddParticipant(ParticipantLog{EntityID: testPlayerEntity, Name: "Alice", Class: "gunner"})
	s.AddParticipant(ParticipantLog{EntityID: "enemy_1000", Name: "Boss", Class: "boss"})

	s.Finalize(OutcomePlayerWin, 6000)

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("got %d instances, want 1", len(instances))
	}

	inst := instances[0]
	if len(inst.Participants) != 2 {
		t.Fatalf("participants = %d, want 2", len(inst.Participants))
	}
	// AddParticipant should set InstanceID
	if inst.Participants[0].InstanceID != "inst-1" {
		t.Errorf("participant InstanceID = %s, want inst-1", inst.Participants[0].InstanceID)
	}
}

func TestSession_Finalize(t *testing.T) {
	sink := NewInMemorySink()
	s := NewSession(sink, "inst-1", "group-1", "arena_boss", "arena", "run-1", 0, SourceSimulation, 100)

	s.Finalize(OutcomeBossWin, 6100)

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("got %d instances, want 1", len(instances))
	}

	inst := instances[0]
	if inst.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %s", inst.InstanceID)
	}
	if inst.Outcome != OutcomeBossWin {
		t.Errorf("Outcome = %s, want boss_win", inst.Outcome)
	}
	if inst.Source != SourceSimulation {
		t.Errorf("Source = %s, want simulation", inst.Source)
	}
	// (6100-100) * 50ms = 300s
	wantDur := 300 * time.Second
	if inst.Duration != wantDur {
		t.Errorf("Duration = %v, want %v", inst.Duration, wantDur)
	}
}
