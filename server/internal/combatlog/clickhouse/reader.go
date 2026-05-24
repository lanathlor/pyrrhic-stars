package clickhouse

import (
	"context"
	"database/sql"
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
		inst, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, inst)
	}
	return results, nil
}

func scanInstance(rows interface{ Scan(dest ...any) error }) (combatlog.InstanceLog, error) {
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
		return combatlog.InstanceLog{}, fmt.Errorf("scan instance: %w", err)
	}
	inst.MobGroupID = int(mobGroupID)
	inst.StartedAt = startedAt
	inst.Duration = time.Duration(durationMS) * time.Millisecond
	inst.Outcome = combatlog.Outcome(outcome)
	inst.Source = combatlog.LogSource(source)
	return inst, nil
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
		if err == io.EOF || errors.Is(err, sql.ErrNoRows) {
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

// ListParticipantsByFilter returns participants for instances matching the
// given filter, using a subquery to avoid exceeding ClickHouse's max query size.
func (r *Repo) ListParticipantsByFilter(ctx context.Context, filter combatlog.InstanceFilter) (map[string][]combatlog.ParticipantLog, error) {
	subquery, subArgs := buildInstanceSubquery(filter)

	query := fmt.Sprintf(
		`SELECT instance_id, entity_id, name, class, is_bot, bot_profile
		FROM participants WHERE instance_id IN (%s)`, subquery)

	rows, err := r.conn.Query(ctx, query, subArgs...)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable

	result := make(map[string][]combatlog.ParticipantLog)
	for rows.Next() {
		var p combatlog.ParticipantLog
		if err := rows.Scan(&p.InstanceID, &p.EntityID, &p.Name, &p.Class, &p.IsBot, &p.BotProfile); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		result[p.InstanceID] = append(result[p.InstanceID], p)
	}
	return result, nil
}

// GetEncounterStats runs aggregate queries across matching instances and
// returns per-instance combat stats plus encounter-wide boss ability stats.
// Uses subqueries against the instances table to avoid exceeding ClickHouse's
// max query size when there are thousands of instances.
func (r *Repo) GetEncounterStats(ctx context.Context, filter combatlog.InstanceFilter) (*combatlog.EncounterStats, error) {
	subquery, subArgs := buildInstanceSubquery(filter)

	stats := &combatlog.EncounterStats{
		InstanceDamage:  make(map[string]map[string]float32),
		InstanceHealing: make(map[string]map[string]float32),
		InstanceDeaths:  make(map[string]int),
		InstancePhases:  make(map[string]string),
	}

	if err := r.loadInstanceClassStats(ctx, stats, subquery, subArgs); err != nil {
		return nil, err
	}
	if err := r.loadInstanceDeaths(ctx, stats, subquery, subArgs); err != nil {
		return nil, err
	}
	if err := r.loadInstancePhases(ctx, stats, subquery, subArgs); err != nil {
		return nil, err
	}
	if err := r.loadBossAbilityStats(ctx, stats, subquery, subArgs); err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *Repo) loadInstanceClassStats(ctx context.Context, stats *combatlog.EncounterStats, subquery string, subArgs []any) error {
	// Per-instance class damage (players -> boss).
	q := fmt.Sprintf(
		`SELECT instance_id, source_class, SUM(amount) as total
		FROM combat_events
		WHERE instance_id IN (%s) AND event_type = 1 AND source LIKE 'player_%%'
		GROUP BY instance_id, source_class`, subquery)
	rows, err := r.conn.Query(ctx, q, subArgs...)
	if err != nil {
		return fmt.Errorf("get encounter stats damage: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable
	for rows.Next() {
		var instID, class string
		var total float64
		if err := rows.Scan(&instID, &class, &total); err != nil {
			return fmt.Errorf("scan encounter stats damage: %w", err)
		}
		if stats.InstanceDamage[instID] == nil {
			stats.InstanceDamage[instID] = make(map[string]float32)
		}
		stats.InstanceDamage[instID][class] = float32(total)
	}

	// Per-instance class healing.
	q = fmt.Sprintf(
		`SELECT instance_id, source_class, SUM(amount) as total
		FROM combat_events
		WHERE instance_id IN (%s) AND event_type = 2 AND source LIKE 'player_%%'
		GROUP BY instance_id, source_class`, subquery)
	rows2, err := r.conn.Query(ctx, q, subArgs...)
	if err != nil {
		return fmt.Errorf("get encounter stats healing: %w", err)
	}
	defer rows2.Close() //nolint:errcheck // rows.Close error is not actionable
	for rows2.Next() {
		var instID, class string
		var total float64
		if err := rows2.Scan(&instID, &class, &total); err != nil {
			return fmt.Errorf("scan encounter stats healing: %w", err)
		}
		if stats.InstanceHealing[instID] == nil {
			stats.InstanceHealing[instID] = make(map[string]float32)
		}
		stats.InstanceHealing[instID][class] = float32(total)
	}
	return nil
}

func (r *Repo) loadInstanceDeaths(ctx context.Context, stats *combatlog.EncounterStats, subquery string, subArgs []any) error {
	q := fmt.Sprintf(
		`SELECT instance_id, COUNT(*) as death_count
		FROM combat_events
		WHERE instance_id IN (%s) AND event_type = 11 AND target LIKE 'player_%%'
		GROUP BY instance_id`, subquery)
	rows, err := r.conn.Query(ctx, q, subArgs...)
	if err != nil {
		return fmt.Errorf("get encounter stats deaths: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable
	for rows.Next() {
		var instID string
		var count uint64
		if err := rows.Scan(&instID, &count); err != nil {
			return fmt.Errorf("scan encounter stats deaths: %w", err)
		}
		stats.InstanceDeaths[instID] = int(count)
	}
	return nil
}

func (r *Repo) loadInstancePhases(ctx context.Context, stats *combatlog.EncounterStats, subquery string, subArgs []any) error {
	q := fmt.Sprintf(
		`SELECT instance_id, max(phase) as max_phase
		FROM combat_events
		WHERE instance_id IN (%s) AND phase != ''
		GROUP BY instance_id`, subquery)
	rows, err := r.conn.Query(ctx, q, subArgs...)
	if err != nil {
		return fmt.Errorf("get encounter stats phases: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable
	for rows.Next() {
		var instID, maxPhase string
		if err := rows.Scan(&instID, &maxPhase); err != nil {
			return fmt.Errorf("scan encounter stats phases: %w", err)
		}
		stats.InstancePhases[instID] = maxPhase
	}
	return nil
}

func (r *Repo) loadBossAbilityStats(ctx context.Context, stats *combatlog.EncounterStats, subquery string, subArgs []any) error {
	q := fmt.Sprintf(
		`SELECT ability_id,
			SUM(CASE WHEN event_type = 1 THEN amount ELSE 0 END) as total_damage,
			countIf(event_type = 1) as hits,
			countIf(event_type = 11) as kills,
			countIf(event_type = 10) as dodges
		FROM combat_events
		WHERE instance_id IN (%s) AND source LIKE 'enemy_%%' AND event_type IN (1, 10, 11)
		GROUP BY ability_id
		ORDER BY total_damage DESC`, subquery)
	rows, err := r.conn.Query(ctx, q, subArgs...)
	if err != nil {
		return fmt.Errorf("get encounter stats boss abilities: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is not actionable
	for rows.Next() {
		var ab combatlog.BossAbilityStat
		var totalDmg float64
		var hits, kills, dodges uint64
		if err := rows.Scan(&ab.AbilityID, &totalDmg, &hits, &kills, &dodges); err != nil {
			return fmt.Errorf("scan encounter stats boss abilities: %w", err)
		}
		ab.TotalDamage = float32(totalDmg)
		ab.Hits = int(hits)
		ab.Kills = int(kills)
		ab.Dodges = int(dodges)
		stats.BossAbilities = append(stats.BossAbilities, ab)
	}
	return nil
}

// buildInstanceSubquery returns a SELECT subquery and its args that resolves
// instance IDs matching the given filter, avoiding huge IN-clause literals.
func buildInstanceSubquery(filter combatlog.InstanceFilter) (string, []any) {
	q := `SELECT instance_id FROM instances WHERE 1=1`
	var args []any

	if filter.EncounterID != "" {
		q += " AND encounter_id = ?"
		args = append(args, filter.EncounterID)
	}
	if filter.Source != "" {
		q += " AND source = ?"
		args = append(args, filter.Source)
	}
	if filter.Outcome != "" {
		q += " AND outcome = ?"
		args = append(args, filter.Outcome)
	}
	if filter.GroupID != "" {
		q += " AND group_id = ?"
		args = append(args, filter.GroupID)
	}

	if filter.Limit > 0 {
		q += " ORDER BY started_at DESC LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			q += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	return q, args
}

// GetReplay returns the recorded WorldState frames for an instance.
func (r *Repo) GetReplay(ctx context.Context, instanceID string) ([][]byte, error) {
	row := r.conn.QueryRow(ctx,
		`SELECT data FROM replay_data WHERE instance_id = ?`, instanceID)

	var encoded string
	if err := row.Scan(&encoded); err != nil {
		if err == io.EOF || errors.Is(err, sql.ErrNoRows) {
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
