package combatlog

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ReadRepository is the query interface for combat log data.
type ReadRepository interface {
	ListInstances(ctx context.Context, filter InstanceFilter) ([]InstanceLog, error)
	GetInstance(ctx context.Context, instanceID string) (*InstanceLog, error)
	GetEvents(ctx context.Context, instanceID string, filter EventFilter) ([]LogEntry, error)
	GetReplay(ctx context.Context, instanceID string) ([][]byte, error)
	ListParticipants(ctx context.Context, instanceIDs []string) (map[string][]ParticipantLog, error)
}

// InstanceFilter controls which instances are returned.
type InstanceFilter struct {
	GroupID     string
	EncounterID string
	Outcome     string
	Source      string
	Limit       int
	Offset      int
}

// EventFilter controls which events are returned.
type EventFilter struct {
	Source string
	Type   string
	Phase  string
}
