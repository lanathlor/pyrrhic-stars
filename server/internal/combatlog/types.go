package combatlog

import "time"

// EventType identifies the kind of combat event.
type EventType uint8

const (
	EventDamage        EventType = 1
	EventHeal          EventType = 2
	EventBuffApply     EventType = 3
	EventBuffRemove    EventType = 4
	EventBuffTick      EventType = 5
	EventCastStart     EventType = 6
	EventCastEnd       EventType = 7
	EventCooldownStart EventType = 8
	EventCooldownEnd   EventType = 9
	EventDodge         EventType = 10
	EventDeath         EventType = 11
	EventPhaseChange   EventType = 12
)

// Outcome describes how an encounter ended.
type Outcome string

const (
	OutcomePlayerWin Outcome = "player_win"
	OutcomeBossWin   Outcome = "boss_win"
	OutcomeTimeout   Outcome = "timeout"
)

// LogSource identifies where the combat data came from.
type LogSource string

const (
	SourceSimulation LogSource = "simulation"
	SourceLive       LogSource = "live"
)

// LogEntry is a single combat event. All fields are value types so entries
// can be safely passed through channels without aliasing.
type LogEntry struct {
	GroupID       string        `json:"group_id"`
	InstanceID    string        `json:"instance_id"`
	EncounterID   string        `json:"encounter_id"`
	RunID         string        `json:"run_id"`
	MobGroupID    int           `json:"mob_group_id"`
	Tick          int           `json:"tick"`
	Timestamp     time.Duration `json:"timestamp_ms"`
	SourceEntity  string        `json:"source"`
	SourceClass   string        `json:"source_class"`
	Target        string        `json:"target"`
	EventType     EventType     `json:"event_type"`
	AbilityID     string        `json:"ability_id"`
	Amount        float32       `json:"amount"`
	Overkill      float32       `json:"overkill"`
	School        string        `json:"school"`
	IsCrit        bool          `json:"is_crit"`
	IsDodged      bool          `json:"is_dodged"`
	Phase         string        `json:"phase"`
	BossHealth    float32       `json:"boss_health"`
	PosX          float32       `json:"pos_x"`
	PosY          float32       `json:"pos_y"`
	PosZ          float32       `json:"pos_z"`
	ResourceType  string        `json:"resource_type"`
	ResourceDelta float32       `json:"resource_delta"`
	ResourceAfter float32       `json:"resource_after"`
}

// InstanceLog records metadata about a complete encounter run.
type InstanceLog struct {
	InstanceID   string           `json:"instance_id"`
	GroupID      string           `json:"group_id"`     // player party ID
	EncounterID  string           `json:"encounter_id"` // human-readable name (e.g. "guard_captain")
	ZoneID       string           `json:"zone_id"`
	RunID        string           `json:"run_id"`
	MobGroupID   int              `json:"mob_group_id"` // enemy pack GroupID, or enemy ID for solo mobs
	StartedAt    time.Time        `json:"started_at"`
	Duration     time.Duration    `json:"duration_ms"`
	Outcome      Outcome          `json:"outcome"`
	Source       LogSource        `json:"source"`
	Participants []ParticipantLog `json:"participants"`
}

// ParticipantLog identifies one entity that participated in an encounter.
type ParticipantLog struct {
	InstanceID string `json:"instance_id"`
	EntityID   string `json:"entity_id"`
	Name       string `json:"name"`
	Class      string `json:"class"`
	IsBot      bool   `json:"is_bot"`
	BotProfile string `json:"bot_profile,omitempty"`
}
