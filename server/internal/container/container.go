package container

import (
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/persistence"
)

// Container holds all injectable dependencies for the application.
type Container struct {
	Repo          persistence.Repository
	CombatLogSink combatlog.EventSink
}

// New creates a container with the given dependencies.
func New(repo persistence.Repository) *Container {
	return &Container{
		Repo:          repo,
		CombatLogSink: combatlog.NullSink{},
	}
}
