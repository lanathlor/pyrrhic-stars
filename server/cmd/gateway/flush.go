package main

import (
	"context"
	"log/slog"
	"time"

	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

// savePlayerPosition snapshots a player's hub position to the database.
func (g *gateway) savePlayerPosition(sess *session.Session) {
	zi := g.getZone(zone.ZoneHub)
	if zi == nil {
		return
	}
	p := zi.zone.GetPlayer(sess.PeerID)
	if p == nil {
		return
	}
	if sess.CharID == 0 {
		return
	}
	if err := g.container.Repo.UpdateCharacterPosition(
		sess.CharID,
		float64(p.Position.X),
		float64(p.Position.Y),
		float64(p.Position.Z),
		float64(p.RotationY),
	); err != nil {
		slog.Error("save player position", "uuid", sess.UserUUID, "error", err)
	}
}

// periodicFlush saves all hub player positions every 30 seconds.
func (g *gateway) periodicFlush(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			g.flushAllPositions()
		case <-ctx.Done():
			return
		}
	}
}

func (g *gateway) flushAllPositions() {
	targets := g.sessions.HubFlushTargets()
	if len(targets) == 0 {
		return
	}

	hubZI := g.getZone(zone.ZoneHub)
	if hubZI == nil {
		return
	}

	saved := 0
	for _, t := range targets {
		p := hubZI.zone.GetPlayer(t.PeerID)
		if p == nil {
			continue
		}
		if err := g.container.Repo.UpdateCharacterPosition(
			t.CharID,
			float64(p.Position.X),
			float64(p.Position.Y),
			float64(p.Position.Z),
			float64(p.RotationY),
		); err != nil {
			slog.Error("periodic flush", "uuid", t.UserUUID, "error", err)
		} else {
			saved++
		}
	}
	if saved > 0 {
		slog.Debug("periodic flush completed", "saved", saved)
	}
}
