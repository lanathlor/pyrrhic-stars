package combatlog

import "sync"

// EventSink is the interface for combat event consumers.
type EventSink interface {
	Log(LogEntry)
	LogInstance(InstanceLog)
	LogReplay(instanceID string, frames [][]byte)
	Close() error
}

// NullSink discards all events. Zero allocation, zero overhead.
type NullSink struct{}

func (NullSink) Log(LogEntry)               {}
func (NullSink) LogInstance(InstanceLog)    {}
func (NullSink) LogReplay(string, [][]byte) {}
func (NullSink) Close() error               { return nil }

// InMemorySink collects events in slices for testing.
type InMemorySink struct {
	mu         sync.Mutex
	events     []LogEntry
	instances  []InstanceLog
	replayData map[string][][]byte // instanceID -> frames
}

func NewInMemorySink() *InMemorySink {
	return &InMemorySink{}
}

func (s *InMemorySink) Log(e LogEntry) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *InMemorySink) LogInstance(inst InstanceLog) {
	s.mu.Lock()
	s.instances = append(s.instances, inst)
	s.mu.Unlock()
}

func (s *InMemorySink) LogReplay(instanceID string, frames [][]byte) {
	s.mu.Lock()
	if s.replayData == nil {
		s.replayData = make(map[string][][]byte)
	}
	s.replayData[instanceID] = frames
	s.mu.Unlock()
}

func (s *InMemorySink) Close() error { return nil }

// ReplayFrames returns the recorded replay frames for an instance.
func (s *InMemorySink) ReplayFrames(instanceID string) [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.replayData[instanceID]
}

// Events returns a copy of all logged events.
func (s *InMemorySink) Events() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LogEntry, len(s.events))
	copy(out, s.events)
	return out
}

// EventsOfType returns events matching the given type.
func (s *InMemorySink) EventsOfType(t EventType) []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []LogEntry
	for _, e := range s.events {
		if e.EventType == t {
			out = append(out, e)
		}
	}
	return out
}

// Instances returns a copy of all logged instance records.
func (s *InMemorySink) Instances() []InstanceLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]InstanceLog, len(s.instances))
	copy(out, s.instances)
	return out
}
