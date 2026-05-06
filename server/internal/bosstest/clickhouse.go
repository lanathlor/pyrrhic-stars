package bosstest

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"codex-online/server/internal/combatlog"
	chrepo "codex-online/server/internal/combatlog/clickhouse"

	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseSink is a synchronous, buffered EventSink for batch simulation
// workloads. Events are collected in memory and flushed to ClickHouse on Close().
// This avoids the async Logger's non-blocking channel which silently drops
// events when simulations produce them faster than ClickHouse can ingest.
type ClickHouseSink struct {
	repo      combatlog.Repository
	mu        sync.Mutex
	events    []combatlog.LogEntry
	instances []combatlog.InstanceLog
	replays   []replayEntry
}

type replayEntry struct {
	instanceID string
	frames     [][]byte
}

func (s *ClickHouseSink) Log(e combatlog.LogEntry) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *ClickHouseSink) LogInstance(inst combatlog.InstanceLog) {
	s.mu.Lock()
	s.instances = append(s.instances, inst)
	s.mu.Unlock()
}

func (s *ClickHouseSink) LogReplay(instanceID string, frames [][]byte) {
	s.mu.Lock()
	s.replays = append(s.replays, replayEntry{instanceID, frames})
	s.mu.Unlock()
}

// Close flushes all buffered data to ClickHouse and closes the connection.
func (s *ClickHouseSink) Close() error {
	s.mu.Lock()
	events := s.events
	instances := s.instances
	replays := s.replays
	s.events = nil
	s.instances = nil
	s.replays = nil
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var errs []error

	// Flush events in batches of 1000
	for i := 0; i < len(events); i += 1000 {
		end := min(i+1000, len(events))
		if err := s.repo.InsertEvents(ctx, events[i:end]); err != nil {
			errs = append(errs, fmt.Errorf("insert events batch %d-%d: %w", i, end, err))
		}
	}

	for _, inst := range instances {
		if err := s.repo.InsertInstance(ctx, inst); err != nil {
			errs = append(errs, fmt.Errorf("insert instance %s: %w", inst.InstanceID, err))
		}
	}

	for _, r := range replays {
		if err := s.repo.InsertReplay(ctx, r.instanceID, r.frames); err != nil {
			errs = append(errs, fmt.Errorf("insert replay %s: %w", r.instanceID, err))
		}
	}

	closeErr := s.repo.Close()

	if len(errs) > 0 {
		for _, e := range errs {
			slog.Error("bosstest: clickhouse flush", "error", e)
		}
		return fmt.Errorf("clickhouse flush: %d errors (first: %w)", len(errs), errs[0])
	}

	slog.Info("bosstest: clickhouse flush complete",
		"events", len(events), "instances", len(instances), "replays", len(replays))

	return closeErr
}

// TryClickHouseSink attempts to connect to ClickHouse using environment variables.
// Returns nil if unavailable (tests fall back to InMemorySink).
func TryClickHouseSink() *ClickHouseSink {
	addr := os.Getenv("CLICKHOUSE_ADDR")
	if addr == "" {
		return nil
	}

	db := os.Getenv("CLICKHOUSE_DB")
	if db == "" {
		db = "codex"
	}
	user := os.Getenv("CLICKHOUSE_USER")
	if user == "" {
		user = "codex"
	}
	pass := os.Getenv("CLICKHOUSE_PASSWORD")

	conn, err := clickhousedriver.Open(&clickhousedriver.Options{
		Addr: []string{addr},
		Auth: clickhousedriver.Auth{
			Database: db,
			Username: user,
			Password: pass,
		},
	})
	if err != nil {
		slog.Warn("bosstest: clickhouse connect failed, using in-memory", "error", err)
		return nil
	}

	ctx := context.Background()
	if err := chrepo.EnsureSchema(ctx, conn); err != nil {
		slog.Warn("bosstest: clickhouse schema failed, using in-memory", "error", err)
		_ = conn.Close()
		return nil
	}

	repo := chrepo.NewRepo(conn)
	return &ClickHouseSink{repo: repo}
}
