package combatlog

import (
	"fmt"
	"time"
)

// EncounterSession tracks per-fight state and provides helpers to emit events
// with pre-filled context fields. Created when a fight starts, finalized when
// it ends.
type EncounterSession struct {
	sink         EventSink
	instanceID   string
	groupID      string
	encounterID  string
	zoneID       string
	runID        string
	mobGroupID   int // enemy pack GroupID, or enemy ID for solo mobs
	source       LogSource
	startTick    uint32
	startTime    time.Time
	participants []ParticipantLog
	lastPhase    map[uint16]int // enemy ID → last known phase

	// Recorder captures binary WorldState frames for replay.
	Recorder *ReplayRecorder
}

// NewSession creates a new encounter session. The sink receives all events.
// mobGroupID is the enemy pack GroupID (positive) or solo enemy ID (positive, from -synthetic key).
func NewSession(sink EventSink, instanceID, groupID, encounterID, zoneID, runID string, mobGroupID int, source LogSource, startTick uint32) *EncounterSession {
	return &EncounterSession{
		sink:        sink,
		instanceID:  instanceID,
		groupID:     groupID,
		encounterID: encounterID,
		zoneID:      zoneID,
		runID:       runID,
		mobGroupID:  mobGroupID,
		source:      source,
		startTick:   startTick,
		startTime:   time.Now(),
		lastPhase:   make(map[uint16]int),
		Recorder:    NewReplayRecorder(),
	}
}

// AddParticipant registers an entity as a fight participant.
func (s *EncounterSession) AddParticipant(p ParticipantLog) {
	p.InstanceID = s.instanceID
	s.participants = append(s.participants, p)
}

// LogEvent emits a combat event, auto-filling encounter context fields.
// bossHealth is the boss HP ratio (0-1), bossPhase is the current phase number.
func (s *EncounterSession) LogEvent(tick uint32, bossHealth float32, bossPhase int, entry LogEntry) {
	entry.GroupID = s.groupID
	entry.InstanceID = s.instanceID
	entry.EncounterID = s.encounterID
	entry.RunID = s.runID
	entry.MobGroupID = s.mobGroupID
	entry.Tick = int(tick)
	entry.Timestamp = time.Duration(tick-s.startTick) * 50 * time.Millisecond
	entry.BossHealth = bossHealth
	entry.Phase = fmt.Sprintf("phase_%d", bossPhase)
	s.sink.Log(entry)
}

// CheckPhaseChange detects and logs phase transitions for an enemy.
// Call after dealing damage to an enemy.
func (s *EncounterSession) CheckPhaseChange(tick uint32, enemyID uint16, currentPhase int, bossHealth float32) {
	prev, ok := s.lastPhase[enemyID]
	if ok && prev != currentPhase {
		s.LogEvent(tick, bossHealth, currentPhase, LogEntry{
			EventType: EventPhaseChange,
			Target:    FormatEnemyID(enemyID),
			Phase:     fmt.Sprintf("phase_%d", currentPhase),
		})
	}
	s.lastPhase[enemyID] = currentPhase
}

// Finalize computes the encounter duration, writes the InstanceLog and replay
// data, and marks the session as complete.
func (s *EncounterSession) Finalize(outcome Outcome, endTick uint32) {
	duration := time.Duration(endTick-s.startTick) * 50 * time.Millisecond
	s.sink.LogInstance(InstanceLog{
		InstanceID:   s.instanceID,
		GroupID:      s.groupID,
		EncounterID:  s.encounterID,
		ZoneID:       s.zoneID,
		RunID:        s.runID,
		MobGroupID:   s.mobGroupID,
		StartedAt:    s.startTime,
		Duration:     duration,
		Outcome:      outcome,
		Source:       s.source,
		Participants: s.participants,
	})
	if s.Recorder != nil && s.Recorder.FrameCount() > 0 {
		s.sink.LogReplay(s.instanceID, s.Recorder.Frames())
	}
}
