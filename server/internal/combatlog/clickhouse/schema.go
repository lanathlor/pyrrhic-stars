package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// DDL statements for the combat log tables.
const (
	CreateInstances = `
CREATE TABLE IF NOT EXISTS instances (
    instance_id  String,
    group_id     String,
    encounter_id String,
    zone_id      String DEFAULT '',
    run_id       String DEFAULT '',
    mob_group_id Int32 DEFAULT 0,
    started_at   DateTime,
    duration_ms  UInt32,
    outcome      LowCardinality(String),
    source       LowCardinality(String),
    created_at   DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (encounter_id, group_id, instance_id)`

	CreateParticipants = `
CREATE TABLE IF NOT EXISTS participants (
    instance_id String,
    entity_id   String,
    name        String,
    class       LowCardinality(String),
    is_bot      Bool DEFAULT false,
    bot_profile String DEFAULT ''
) ENGINE = MergeTree()
ORDER BY (instance_id, entity_id)`

	CreateCombatEvents = `
CREATE TABLE IF NOT EXISTS combat_events (
    instance_id    String,
    group_id       String,
    encounter_id   LowCardinality(String),
    run_id         String DEFAULT '',
    mob_group_id   Int32 DEFAULT 0,
    tick           UInt32,
    timestamp_ms   UInt32,
    source         String,
    source_class   LowCardinality(String),
    target         String,
    event_type     UInt8,
    ability_id     LowCardinality(String),
    amount         Float32 DEFAULT 0,
    overkill       Float32 DEFAULT 0,
    school         LowCardinality(String) DEFAULT '',
    is_crit        Bool DEFAULT false,
    is_dodged      Bool DEFAULT false,
    phase          LowCardinality(String) DEFAULT '',
    boss_health    Float32 DEFAULT 0,
    pos_x          Float32 DEFAULT 0,
    pos_y          Float32 DEFAULT 0,
    pos_z          Float32 DEFAULT 0,
    resource_type  LowCardinality(String) DEFAULT '',
    resource_delta Float32 DEFAULT 0,
    resource_after Float32 DEFAULT 0
) ENGINE = MergeTree()
ORDER BY (encounter_id, group_id, instance_id, tick)`

	CreateReplayData = `
CREATE TABLE IF NOT EXISTS replay_data (
    instance_id  String,
    frame_count  UInt32,
    data         String,
    created_at   DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY instance_id`
)

// Migrations add columns that may be missing from tables created before
// the replay/per-group encounter changes. ClickHouse ignores duplicate
// ADD COLUMN if the column already exists (no-op).
var migrations = []string{
	`ALTER TABLE instances ADD COLUMN IF NOT EXISTS zone_id String DEFAULT ''`,
	`ALTER TABLE instances ADD COLUMN IF NOT EXISTS run_id String DEFAULT ''`,
	`ALTER TABLE instances ADD COLUMN IF NOT EXISTS mob_group_id Int32 DEFAULT 0`,
	`ALTER TABLE combat_events ADD COLUMN IF NOT EXISTS run_id String DEFAULT ''`,
	`ALTER TABLE combat_events ADD COLUMN IF NOT EXISTS mob_group_id Int32 DEFAULT 0`,
}

// EnsureSchema creates all combat log tables if they don't exist and
// applies any column migrations for pre-existing tables.
func EnsureSchema(ctx context.Context, conn driver.Conn) error {
	for _, ddl := range []string{CreateInstances, CreateParticipants, CreateCombatEvents, CreateReplayData} {
		if err := conn.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("combatlog schema: %w", err)
		}
	}
	for _, m := range migrations {
		if err := conn.Exec(ctx, m); err != nil {
			return fmt.Errorf("combatlog migration: %w", err)
		}
	}
	return nil
}
