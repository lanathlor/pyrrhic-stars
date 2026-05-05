package clickhouse

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"codex-online/server/internal/combatlog"
)

// ListInstances queries instance records with optional filtering.
func (r *Repo) ListInstances(ctx context.Context, filter combatlog.InstanceFilter) ([]combatlog.InstanceLog, error) {
	query := `SELECT instance_id, group_id, encounter_id, zone_id, run_id, mob_group_id, started_at, duration_ms, outcome, source
		FROM instances WHERE 1=1`
	args := make([]any, 0, 4)

	if filter.GroupID != "" {
		query += " AND group_id = ?"
		args = append(args, filter.GroupID)
	}
	if filter.EncounterID != "" {
		query += " AND encounter_id = ?"
		args = append(args, filter.EncounterID)
	}
	if filter.Outcome != "" {
		query += " AND outcome = ?"
		args = append(args, filter.Outcome)
	}
	if filter.Source != "" {
		query += " AND source = ?"
		args = append(args, filter.Source)
	}

	query += " ORDER BY started_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ?"
	args = append(args, limit)
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable

	var results []combatlog.InstanceLog
	for rows.Next() {
		var inst combatlog.InstanceLog
		var startedAt time.Time
		var durationMS uint32
		var outcome, source string

		var mobGroupID int32
		if err := rows.Scan(
			&inst.InstanceID, &inst.GroupID, &inst.EncounterID,
			&inst.ZoneID, &inst.RunID, &mobGroupID,
			&startedAt, &durationMS, &outcome, &source,
		); err != nil {
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		inst.MobGroupID = int(mobGroupID)
		inst.StartedAt = startedAt
		inst.Duration = time.Duration(durationMS) * time.Millisecond
		inst.Outcome = combatlog.Outcome(outcome)
		inst.Source = combatlog.LogSource(source)
		results = append(results, inst)
	}
	return results, nil
}

// GetInstance returns a single instance with its participants.
func (r *Repo) GetInstance(ctx context.Context, instanceID string) (*combatlog.InstanceLog, error) {
	row := r.conn.QueryRow(ctx,
		`SELECT instance_id, group_id, encounter_id, zone_id, run_id, mob_group_id, started_at, duration_ms, outcome, source
		FROM instances WHERE instance_id = ?`, instanceID)

	var inst combatlog.InstanceLog
	var startedAt time.Time
	var durationMS uint32
	var mobGroupID int32
	var outcome, source string

	if err := row.Scan(
		&inst.InstanceID, &inst.GroupID, &inst.EncounterID,
		&inst.ZoneID, &inst.RunID, &mobGroupID,
		&startedAt, &durationMS, &outcome, &source,
	); err != nil {
		if err == io.EOF {
			return nil, combatlog.ErrNotFound
		}
		return nil, fmt.Errorf("get instance: %w", err)
	}
	inst.MobGroupID = int(mobGroupID)
	inst.StartedAt = startedAt
	inst.Duration = time.Duration(durationMS) * time.Millisecond
	inst.Outcome = combatlog.Outcome(outcome)
	inst.Source = combatlog.LogSource(source)

	// Load participants.
	rows, err := r.conn.Query(ctx,
		`SELECT instance_id, entity_id, name, class, is_bot, bot_profile
		FROM participants WHERE instance_id = ?`, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get participants: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable

	for rows.Next() {
		var p combatlog.ParticipantLog
		if err := rows.Scan(&p.InstanceID, &p.EntityID, &p.Name, &p.Class, &p.IsBot, &p.BotProfile); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		inst.Participants = append(inst.Participants, p)
	}

	return &inst, nil
}

// GetEvents returns combat events for an instance, ordered by tick.
func (r *Repo) GetEvents(ctx context.Context, instanceID string, filter combatlog.EventFilter) ([]combatlog.LogEntry, error) {
	query := `SELECT
		instance_id, group_id, encounter_id, run_id, tick, timestamp_ms,
		source, source_class, target, event_type, ability_id,
		amount, overkill, school, is_crit, is_dodged,
		phase, boss_health, pos_x, pos_y, pos_z,
		resource_type, resource_delta, resource_after
		FROM combat_events WHERE instance_id = ?`
	args := []any{instanceID}

	if filter.Source != "" {
		query += " AND source = ?"
		args = append(args, filter.Source)
	}
	if filter.Type != "" {
		query += " AND event_type = ?"
		args = append(args, filter.Type)
	}
	if filter.Phase != "" {
		query += " AND phase = ?"
		args = append(args, filter.Phase)
	}

	query += " ORDER BY tick ASC"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable

	var results []combatlog.LogEntry
	for rows.Next() {
		var e combatlog.LogEntry
		var tick, tsMS uint32
		var eventType uint8

		if err := rows.Scan(
			&e.InstanceID, &e.GroupID, &e.EncounterID, &e.RunID,
			&tick, &tsMS,
			&e.SourceEntity, &e.SourceClass, &e.Target,
			&eventType, &e.AbilityID,
			&e.Amount, &e.Overkill, &e.School,
			&e.IsCrit, &e.IsDodged,
			&e.Phase, &e.BossHealth,
			&e.PosX, &e.PosY, &e.PosZ,
			&e.ResourceType, &e.ResourceDelta, &e.ResourceAfter,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Tick = int(tick)
		e.Timestamp = time.Duration(tsMS) * time.Millisecond
		e.EventType = combatlog.EventType(eventType)
		results = append(results, e)
	}
	return results, nil
}

// ListParticipants returns participants grouped by instance ID for a batch of instances.
func (r *Repo) ListParticipants(ctx context.Context, instanceIDs []string) (map[string][]combatlog.ParticipantLog, error) {
	if len(instanceIDs) == 0 {
		return nil, nil
	}

	query := `SELECT instance_id, entity_id, name, class, is_bot, bot_profile
		FROM participants WHERE instance_id IN (?)`

	rows, err := r.conn.Query(ctx, query, instanceIDs)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable

	result := make(map[string][]combatlog.ParticipantLog, len(instanceIDs))
	for rows.Next() {
		var p combatlog.ParticipantLog
		if err := rows.Scan(&p.InstanceID, &p.EntityID, &p.Name, &p.Class, &p.IsBot, &p.BotProfile); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		result[p.InstanceID] = append(result[p.InstanceID], p)
	}
	return result, nil
}

// GetReplay returns the recorded WorldState frames for an instance.
func (r *Repo) GetReplay(ctx context.Context, instanceID string) ([][]byte, error) {
	row := r.conn.QueryRow(ctx,
		`SELECT data FROM replay_data WHERE instance_id = ?`, instanceID)

	var encoded string
	if err := row.Scan(&encoded); err != nil {
		if err == io.EOF {
			return nil, combatlog.ErrNotFound
		}
		return nil, fmt.Errorf("get replay: %w", err)
	}

	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode replay base64: %w", err)
	}

	if len(blob) < 4 {
		return nil, errors.New("replay data too short")
	}

	frameCount := binary.BigEndian.Uint32(blob[:4])
	offset := 4
	frames := make([][]byte, 0, frameCount)

	for i := uint32(0); i < frameCount; i++ {
		if offset+2 > len(blob) {
			return nil, fmt.Errorf("replay data truncated at frame %d header", i)
		}
		frameLen := int(binary.BigEndian.Uint16(blob[offset : offset+2]))
		offset += 2
		if offset+frameLen > len(blob) {
			return nil, fmt.Errorf("replay data truncated at frame %d data", i)
		}
		frames = append(frames, blob[offset:offset+frameLen])
		offset += frameLen
	}

	return frames, nil
}
