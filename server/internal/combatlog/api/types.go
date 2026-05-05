package api

import (
	"time"

	"codex-online/server/internal/combatlog"
)

// InstanceListItem is the JSON shape for the instance list endpoint.
// It converts time.Duration to integer milliseconds so clients get a
// usable number instead of Go's nanosecond representation.
type InstanceListItem struct {
	InstanceID   string                     `json:"instance_id"`
	GroupID      string                     `json:"group_id"`
	EncounterID  string                     `json:"encounter_id"`
	StartedAt    time.Time                  `json:"started_at"`
	DurationMS   int                        `json:"duration_ms"`
	Outcome      string                     `json:"outcome"`
	Source       string                     `json:"source"`
	Participants []combatlog.ParticipantLog `json:"participants"`
}

// InstanceDetail is the JSON shape for the single-instance endpoint.
type InstanceDetail struct {
	InstanceID   string                     `json:"instance_id"`
	GroupID      string                     `json:"group_id"`
	EncounterID  string                     `json:"encounter_id"`
	StartedAt    time.Time                  `json:"started_at"`
	DurationMS   int                        `json:"duration_ms"`
	Outcome      string                     `json:"outcome"`
	Source       string                     `json:"source"`
	Participants []combatlog.ParticipantLog `json:"participants"`
}

// EventDTO is the JSON shape for combat events.
// Timestamp is integer milliseconds, not Go's nanosecond time.Duration.
type EventDTO struct {
	GroupID       string  `json:"group_id"`
	InstanceID    string  `json:"instance_id"`
	EncounterID   string  `json:"encounter_id"`
	RunID         string  `json:"run_id"`
	MobGroupID    int     `json:"mob_group_id"`
	Tick          int     `json:"tick"`
	TimestampMS   int     `json:"timestamp_ms"`
	Source        string  `json:"source"`
	SourceClass   string  `json:"source_class"`
	Target        string  `json:"target"`
	EventType     uint8   `json:"event_type"`
	AbilityID     string  `json:"ability_id"`
	Amount        float32 `json:"amount"`
	Overkill      float32 `json:"overkill"`
	School        string  `json:"school"`
	IsCrit        bool    `json:"is_crit"`
	IsDodged      bool    `json:"is_dodged"`
	Phase         string  `json:"phase"`
	BossHealth    float32 `json:"boss_health"`
	PosX          float32 `json:"pos_x"`
	PosY          float32 `json:"pos_y"`
	PosZ          float32 `json:"pos_z"`
	ResourceType  string  `json:"resource_type"`
	ResourceDelta float32 `json:"resource_delta"`
	ResourceAfter float32 `json:"resource_after"`
}

func convertEvents(entries []combatlog.LogEntry) []EventDTO {
	dtos := make([]EventDTO, len(entries))
	for i, e := range entries {
		dtos[i] = EventDTO{
			GroupID:       e.GroupID,
			InstanceID:    e.InstanceID,
			EncounterID:   e.EncounterID,
			RunID:         e.RunID,
			MobGroupID:    e.MobGroupID,
			Tick:          e.Tick,
			TimestampMS:   int(e.Timestamp.Milliseconds()),
			Source:        e.SourceEntity,
			SourceClass:   e.SourceClass,
			Target:        e.Target,
			EventType:     uint8(e.EventType),
			AbilityID:     e.AbilityID,
			Amount:        e.Amount,
			Overkill:      e.Overkill,
			School:        e.School,
			IsCrit:        e.IsCrit,
			IsDodged:      e.IsDodged,
			Phase:         e.Phase,
			BossHealth:    e.BossHealth,
			PosX:          e.PosX,
			PosY:          e.PosY,
			PosZ:          e.PosZ,
			ResourceType:  e.ResourceType,
			ResourceDelta: e.ResourceDelta,
			ResourceAfter: e.ResourceAfter,
		}
	}
	return dtos
}

func convertInstance(inst *combatlog.InstanceLog) InstanceDetail {
	return InstanceDetail{
		InstanceID:   inst.InstanceID,
		GroupID:      inst.GroupID,
		EncounterID:  inst.EncounterID,
		StartedAt:    inst.StartedAt,
		DurationMS:   int(inst.Duration.Milliseconds()),
		Outcome:      string(inst.Outcome),
		Source:       string(inst.Source),
		Participants: inst.Participants,
	}
}

// FightExport is the full fight JSON export structure.
type FightExport struct {
	Version      int                        `json:"version"`
	InstanceID   string                     `json:"instance_id"`
	GroupID      string                     `json:"group_id"`
	EncounterID  string                     `json:"encounter_id"`
	StartTime    time.Time                  `json:"start_time"`
	DurationMS   int                        `json:"duration_ms"`
	Outcome      string                     `json:"outcome"`
	Source       string                     `json:"source"`
	Participants []combatlog.ParticipantLog `json:"participants"`
	Events       []EventDTO                 `json:"events"`
}

// ReplayExport is the response for the replay endpoint.
// Frames are base64-encoded binary WorldState snapshots (one per tick).
// The client decodes each frame with NetSerializer.decode_world_state().
type ReplayExport struct {
	Version      int                        `json:"version"`
	InstanceID   string                     `json:"instance_id"`
	EncounterID  string                     `json:"encounter_id"`
	ZoneID       string                     `json:"zone_id"`
	TickRate     int                        `json:"tick_rate"`
	FrameCount   int                        `json:"frame_count"`
	DurationMS   int                        `json:"duration_ms"`
	Outcome      string                     `json:"outcome"`
	Participants []combatlog.ParticipantLog `json:"participants"`
	Events       []EventDTO                 `json:"events"`
	Frames       []string                   `json:"frames"`
}
