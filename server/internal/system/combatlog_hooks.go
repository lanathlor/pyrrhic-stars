package system

import (
	"log/slog"
	"strconv"
	"time"

	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
)

// enemySessionKey returns the combat log session key for an enemy.
// Pack enemies use their GroupID; solo bosses use a synthetic -enemyID key.
// Non-boss solo enemies (GroupID=0) return 0, meaning no session.
func enemySessionKey(e *entity.Enemy) int {
	if e.GroupID > 0 {
		return e.GroupID
	}
	if e.IsBoss {
		return -int(e.ID)
	}
	return 0
}

// startGroupCombatLog creates an EncounterSession keyed by an enemy group ID.
// Positive keys are enemy pack GroupIDs; negative keys are synthetic IDs for
// solo bosses (-enemyID). Called on first aggro. No-op if already recording.
func startGroupCombatLog(w *World, groupID int) {
	if w.CombatLogSink == nil {
		return
	}
	if w.CombatLogs == nil {
		w.CombatLogs = make(map[int]*combatlog.EncounterSession)
	}
	if _, exists := w.CombatLogs[groupID]; exists {
		return
	}

	// Negative keys are synthetic: -enemyID for solo bosses.
	// Positive keys are enemy pack GroupIDs.
	isSoloBoss := groupID < 0
	matchEnemy := func(e *entity.Enemy) bool {
		if isSoloBoss {
			return int(e.ID) == -groupID
		}
		return e.GroupID == groupID
	}

	// Derive encounter name from the first matching enemy.
	encounterID := w.ZoneID
	for _, e := range w.Enemies {
		if matchEnemy(e) && e.Alive {
			encounterID = e.DefName
			break
		}
	}

	instanceID := w.ZoneID + "_" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + encounterID
	mobID := groupID
	if isSoloBoss {
		mobID = -groupID
	}

	session := combatlog.NewSession(
		w.CombatLogSink, instanceID, w.ZoneID,
		encounterID, w.ZoneID, w.RunID, mobID, combatlog.SourceLive, w.TickNum,
	)

	registerParticipants(w, session, matchEnemy)

	w.CombatLogs[groupID] = session

	slog.Info("combat log started", "zone_id", w.ZoneID, "group_id", groupID, "encounter", encounterID)
}

// registerParticipants adds all alive players and matching enemies to the session.
func registerParticipants(w *World, session *combatlog.EncounterSession, matchEnemy func(*entity.Enemy) bool) {
	for _, p := range w.Players {
		if p.Alive {
			session.AddParticipant(combatlog.ParticipantLog{
				EntityID: combatlog.FormatPlayerID(p.ID),
				Name:     p.Username,
				Class:    p.ClassID,
			})
		}
	}
	for _, e := range w.Enemies {
		if matchEnemy(e) && e.Alive {
			session.AddParticipant(combatlog.ParticipantLog{
				EntityID: combatlog.FormatEnemyID(e.ID),
				Name:     e.DefName,
				Class:    "enemy",
			})
		}
	}
}

// finalizeGroupCombatLog finalizes a single group's encounter session.
func finalizeGroupCombatLog(w *World, groupID int, outcome combatlog.Outcome) {
	if w.CombatLogs == nil {
		return
	}
	session, exists := w.CombatLogs[groupID]
	if !exists {
		return
	}
	session.Finalize(outcome, w.TickNum)
	delete(w.CombatLogs, groupID)
}

// finalizeAllCombatLogs finalizes every active group session (wipe/victory).
func finalizeAllCombatLogs(w *World, outcome combatlog.Outcome) {
	for groupID, session := range w.CombatLogs {
		session.Finalize(outcome, w.TickNum)
		delete(w.CombatLogs, groupID)
	}
}

// checkEnemyGroupDead checks if all enemies sharing a dead enemy's group are
// also dead, and finalizes the group's combat log if so. For solo enemies
// (GroupID=0) that are bosses, the session is finalized immediately.
func checkEnemyGroupDead(w *World, e *entity.Enemy) {
	if e.GroupID > 0 {
		for _, other := range w.Enemies {
			if other.GroupID == e.GroupID && other.Alive {
				return
			}
		}
		finalizeGroupCombatLog(w, e.GroupID, combatlog.OutcomePlayerWin)
	} else if e.IsBoss {
		finalizeGroupCombatLog(w, -int(e.ID), combatlog.OutcomePlayerWin)
	}
}
