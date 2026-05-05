package clickhouse

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"codex-online/server/internal/combatlog"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Repo implements combatlog.Repository using ClickHouse native batch inserts.
type Repo struct {
	conn driver.Conn
}

// NewRepo creates a ClickHouse repository. The caller is responsible for
// creating the connection and calling EnsureSchema before use.
func NewRepo(conn driver.Conn) *Repo {
	return &Repo{conn: conn}
}

// InsertEvents batch-inserts combat events.
func (r *Repo) InsertEvents(ctx context.Context, events []combatlog.LogEntry) error {
	batch, err := r.conn.PrepareBatch(ctx, `INSERT INTO combat_events (
		instance_id, group_id, encounter_id, run_id, mob_group_id, tick, timestamp_ms,
		source, source_class, target, event_type, ability_id,
		amount, overkill, school, is_crit, is_dodged,
		phase, boss_health, pos_x, pos_y, pos_z,
		resource_type, resource_delta, resource_after
	)`)
	if err != nil {
		return fmt.Errorf("prepare combat_events batch: %w", err)
	}
	for _, e := range events {
		if err := batch.Append(
			e.InstanceID, e.GroupID, e.EncounterID, e.RunID, int32(e.MobGroupID),
			uint32(e.Tick), uint32(e.Timestamp.Milliseconds()),
			e.SourceEntity, e.SourceClass, e.Target,
			uint8(e.EventType), e.AbilityID,
			e.Amount, e.Overkill, e.School,
			e.IsCrit, e.IsDodged,
			e.Phase, e.BossHealth,
			e.PosX, e.PosY, e.PosZ,
			e.ResourceType, e.ResourceDelta, e.ResourceAfter,
		); err != nil {
			return fmt.Errorf("append combat_events: %w", err)
		}
	}
	return batch.Send()
}

// InsertInstance inserts an instance record and its participants.
func (r *Repo) InsertInstance(ctx context.Context, inst combatlog.InstanceLog) error {
	// Insert instance metadata
	batch, err := r.conn.PrepareBatch(ctx, `INSERT INTO instances (
		instance_id, group_id, encounter_id, zone_id, run_id, mob_group_id, started_at, duration_ms, outcome, source
	)`)
	if err != nil {
		return fmt.Errorf("prepare instances batch: %w", err)
	}
	if err := batch.Append(
		inst.InstanceID, inst.GroupID, inst.EncounterID, inst.ZoneID, inst.RunID, int32(inst.MobGroupID),
		inst.StartedAt, uint32(inst.Duration.Milliseconds()),
		string(inst.Outcome), string(inst.Source),
	); err != nil {
		return fmt.Errorf("append instances: %w", err)
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send instances: %w", err)
	}

	// Insert participants
	if len(inst.Participants) == 0 {
		return nil
	}
	pBatch, err := r.conn.PrepareBatch(ctx, `INSERT INTO participants (
		instance_id, entity_id, name, class, is_bot, bot_profile
	)`)
	if err != nil {
		return fmt.Errorf("prepare participants batch: %w", err)
	}
	for _, p := range inst.Participants {
		if err := pBatch.Append(
			p.InstanceID, p.EntityID, p.Name, p.Class, p.IsBot, p.BotProfile,
		); err != nil {
			return fmt.Errorf("append participants: %w", err)
		}
	}
	return pBatch.Send()
}

// InsertReplay stores replay frame data as a single base64-encoded binary blob.
// Format: [frame_count:u32][frame_0_len:u16][frame_0_data]...[frame_N_len:u16][frame_N_data]
func (r *Repo) InsertReplay(ctx context.Context, instanceID string, frames [][]byte) error {
	// Build binary blob with length-prefixed frames.
	totalSize := 4 // frame_count header
	for _, f := range frames {
		totalSize += 2 + len(f) // u16 length + data
	}
	blob := make([]byte, 0, totalSize)
	blob = binary.BigEndian.AppendUint32(blob, uint32(len(frames)))
	for _, f := range frames {
		blob = binary.BigEndian.AppendUint16(blob, uint16(len(f)))
		blob = append(blob, f...)
	}

	encoded := base64.StdEncoding.EncodeToString(blob)

	batch, err := r.conn.PrepareBatch(ctx, `INSERT INTO replay_data (instance_id, frame_count, data)`)
	if err != nil {
		return fmt.Errorf("prepare replay_data batch: %w", err)
	}
	if err := batch.Append(instanceID, uint32(len(frames)), encoded); err != nil {
		return fmt.Errorf("append replay_data: %w", err)
	}
	return batch.Send()
}

// Close closes the underlying ClickHouse connection.
func (r *Repo) Close() error {
	return r.conn.Close()
}
