package combatlog

import "testing"

func TestNullSink_NoPanic(t *testing.T) {
	var s NullSink
	s.Log(LogEntry{EventType: EventDamage, Amount: 100})
	s.LogInstance(InstanceLog{InstanceID: "test"})
	if err := s.Close(); err != nil {
		t.Errorf("NullSink.Close() = %v", err)
	}
}

func TestInMemorySink_CollectsEvents(t *testing.T) {
	s := NewInMemorySink()

	s.Log(LogEntry{EventType: EventDamage, Amount: 50})
	s.Log(LogEntry{EventType: EventDeath, Target: testPlayerEntity})
	s.Log(LogEntry{EventType: EventDamage, Amount: 30})

	events := s.Events()
	if len(events) != 3 {
		t.Fatalf("Events() = %d entries, want 3", len(events))
	}
	if events[0].Amount != 50 {
		t.Errorf("events[0].Amount = %f, want 50", events[0].Amount)
	}
}

func TestInMemorySink_EventsOfType(t *testing.T) {
	s := NewInMemorySink()

	s.Log(LogEntry{EventType: EventDamage, Amount: 50})
	s.Log(LogEntry{EventType: EventDeath, Target: testPlayerEntity})
	s.Log(LogEntry{EventType: EventDamage, Amount: 30})
	s.Log(LogEntry{EventType: EventBuffApply, AbilityID: "overclock"})

	dmg := s.EventsOfType(EventDamage)
	if len(dmg) != 2 {
		t.Fatalf("EventsOfType(EventDamage) = %d, want 2", len(dmg))
	}
	deaths := s.EventsOfType(EventDeath)
	if len(deaths) != 1 {
		t.Fatalf("EventsOfType(EventDeath) = %d, want 1", len(deaths))
	}
}

func TestInMemorySink_CollectsInstances(t *testing.T) {
	s := NewInMemorySink()

	s.LogInstance(InstanceLog{InstanceID: "inst1", Outcome: OutcomePlayerWin})
	s.LogInstance(InstanceLog{InstanceID: "inst2", Outcome: OutcomeBossWin})

	instances := s.Instances()
	if len(instances) != 2 {
		t.Fatalf("Instances() = %d, want 2", len(instances))
	}
	if instances[0].Outcome != OutcomePlayerWin {
		t.Errorf("instances[0].Outcome = %s, want player_win", instances[0].Outcome)
	}
}

func TestInMemorySink_ReturnsCopy(t *testing.T) {
	s := NewInMemorySink()
	s.Log(LogEntry{EventType: EventDamage, Amount: 10})

	events1 := s.Events()
	events1[0].Amount = 999

	events2 := s.Events()
	if events2[0].Amount != 10 {
		t.Error("Events() should return a copy, not a reference")
	}
}
