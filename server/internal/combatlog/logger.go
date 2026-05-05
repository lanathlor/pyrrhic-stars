package combatlog

import (
	"context"
	"log/slog"
	"time"
)

// Repository is the storage backend for combat log data.
type Repository interface {
	InsertEvents(ctx context.Context, events []LogEntry) error
	InsertInstance(ctx context.Context, inst InstanceLog) error
	InsertReplay(ctx context.Context, instanceID string, frames [][]byte) error
	Close() error
}

// Logger is a non-blocking combat event logger that batches entries and
// flushes them to a Repository. It implements EventSink.
//
// Log() writes to a buffered channel. If the channel is full, the event is
// silently dropped — the game loop must never block on telemetry.
type Logger struct {
	events        chan LogEntry
	instances     chan InstanceLog
	replays       chan replayMsg
	repo          Repository
	batchSize     int
	flushInterval time.Duration
	done          chan struct{}
	logger        *slog.Logger
}

type replayMsg struct {
	instanceID string
	frames     [][]byte
}

// LoggerOption configures a Logger.
type LoggerOption func(*Logger)

// WithBatchSize sets the number of events per batch flush. Default 1000.
func WithBatchSize(n int) LoggerOption {
	return func(l *Logger) { l.batchSize = n }
}

// WithFlushInterval sets the maximum time between flushes. Default 500ms.
func WithFlushInterval(d time.Duration) LoggerOption {
	return func(l *Logger) { l.flushInterval = d }
}

// WithBufferSize sets the channel buffer capacity. Default 10000.
func WithBufferSize(n int) LoggerOption {
	return func(l *Logger) { l.events = make(chan LogEntry, n) }
}

// WithLogger sets the structured logger for flush errors. Default slog.Default().
func WithLogger(lg *slog.Logger) LoggerOption {
	return func(l *Logger) { l.logger = lg }
}

// NewLogger creates a Logger that flushes batches to repo.
// Call Close() to drain remaining events and shut down.
func NewLogger(repo Repository, opts ...LoggerOption) *Logger {
	l := &Logger{
		events:        make(chan LogEntry, 10000),
		instances:     make(chan InstanceLog, 64),
		replays:       make(chan replayMsg, 16),
		repo:          repo,
		batchSize:     1000,
		flushInterval: 500 * time.Millisecond,
		done:          make(chan struct{}),
		logger:        slog.Default(),
	}
	for _, opt := range opts {
		opt(l)
	}
	go l.run()
	return l
}

// Log enqueues an event. Non-blocking; drops silently if the buffer is full.
func (l *Logger) Log(e LogEntry) {
	select {
	case l.events <- e:
	default:
		// Drop — never block the game loop.
	}
}

// LogInstance enqueues an instance record.
func (l *Logger) LogInstance(inst InstanceLog) {
	select {
	case l.instances <- inst:
	default:
	}
}

// LogReplay enqueues replay frame data for async storage.
func (l *Logger) LogReplay(instanceID string, frames [][]byte) {
	select {
	case l.replays <- replayMsg{instanceID: instanceID, frames: frames}:
	default:
		l.logger.Warn("combatlog.replay_dropped", "instance_id", instanceID, "frames", len(frames))
	}
}

// Close signals the background goroutine to drain and stop, then closes the repo.
func (l *Logger) Close() error {
	close(l.events)
	close(l.instances)
	close(l.replays)
	<-l.done
	return l.repo.Close()
}

func (l *Logger) run() {
	defer close(l.done)

	batch := make([]LogEntry, 0, l.batchSize)
	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := l.repo.InsertEvents(ctx, batch)
		cancel()
		if err != nil {
			l.logger.Error("combatlog.flush", "err", err, "count", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case e, ok := <-l.events:
			if !ok {
				// Channel closed — drain instances and flush remaining.
				flush()
				l.drainInstances()
				return
			}
			batch = append(batch, e)
			if len(batch) >= l.batchSize {
				flush()
			}

		case inst, ok := <-l.instances:
			if !ok {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := l.repo.InsertInstance(ctx, inst); err != nil {
				l.logger.Error("combatlog.flush_instance", "err", err)
			}
			cancel()

		case msg, ok := <-l.replays:
			if !ok {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := l.repo.InsertReplay(ctx, msg.instanceID, msg.frames); err != nil {
				l.logger.Error("combatlog.flush_replay", "err", err, "instance_id", msg.instanceID, "frames", len(msg.frames))
			}
			cancel()

		case <-ticker.C:
			flush()
		}
	}
}

func (l *Logger) drainInstances() {
	for inst := range l.instances {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := l.repo.InsertInstance(ctx, inst); err != nil {
			l.logger.Error("combatlog.flush_instance", "err", err)
		}
		cancel()
	}
	for msg := range l.replays {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := l.repo.InsertReplay(ctx, msg.instanceID, msg.frames); err != nil {
			l.logger.Error("combatlog.flush_replay", "err", err, "instance_id", msg.instanceID)
		}
		cancel()
	}
}
