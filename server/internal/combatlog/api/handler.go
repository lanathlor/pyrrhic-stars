package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"codex-online/server/internal/combatlog"
)

// Handler serves the combat log REST API.
type Handler struct {
	repo combatlog.ReadRepository
}

// NewHandler creates a combat log API handler.
func NewHandler(repo combatlog.ReadRepository) *Handler {
	return &Handler{repo: repo}
}

// Register mounts routes on the given mux under /api/v1/logs/.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/logs/instances", h.listInstances)
	mux.HandleFunc("GET /api/v1/logs/instances/{id}", h.getInstance)
	mux.HandleFunc("GET /api/v1/logs/instances/{id}/events", h.getEvents)
	mux.HandleFunc("GET /api/v1/logs/instances/{id}/export", h.exportInstance)
	mux.HandleFunc("GET /api/v1/logs/instances/{id}/replay", h.getReplay)
	mux.HandleFunc("GET /api/v1/logs/stats/encounter/{encounter_id}", h.getEncounterStats)
}

func (h *Handler) listInstances(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := combatlog.InstanceFilter{
		GroupID:     q.Get("group_id"),
		EncounterID: q.Get("encounter_id"),
		Outcome:     q.Get("outcome"),
		Source:      q.Get("source"),
		Limit:       limit,
		Offset:      offset,
	}

	instances, err := h.repo.ListInstances(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(instances) == 0 {
		writeJSON(w, []InstanceListItem{})
		return
	}

	// Load participants using a subquery to avoid exceeding ClickHouse query size limits.
	partMap, err := h.repo.ListParticipantsByFilter(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response DTOs with duration in milliseconds.
	items := make([]InstanceListItem, len(instances))
	for i, inst := range instances {
		participants := partMap[inst.InstanceID]
		if participants == nil {
			participants = []combatlog.ParticipantLog{}
		}
		items[i] = InstanceListItem{
			InstanceID:   inst.InstanceID,
			GroupID:      inst.GroupID,
			EncounterID:  inst.EncounterID,
			StartedAt:    inst.StartedAt,
			DurationMS:   int(inst.Duration.Milliseconds()),
			Outcome:      string(inst.Outcome),
			Source:       string(inst.Source),
			Participants: participants,
		}
	}

	writeJSON(w, items)
}

func (h *Handler) getInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing instance id", http.StatusBadRequest)
		return
	}

	inst, err := h.repo.GetInstance(r.Context(), id)
	if err != nil {
		if errors.Is(err, combatlog.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, convertInstance(inst))
}

func (h *Handler) getEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing instance id", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	filter := combatlog.EventFilter{
		Source: q.Get("source"),
		Type:   q.Get("type"),
		Phase:  q.Get("phase"),
	}

	events, err := h.repo.GetEvents(r.Context(), id, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []combatlog.LogEntry{}
	}

	writeJSON(w, convertEvents(events))
}

// exportInstance returns the full fight export JSON (instance + participants + all events ordered by tick).
func (h *Handler) exportInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing instance id", http.StatusBadRequest)
		return
	}

	inst, err := h.repo.GetInstance(r.Context(), id)
	if err != nil {
		if errors.Is(err, combatlog.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events, err := h.repo.GetEvents(r.Context(), id, combatlog.EventFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []combatlog.LogEntry{}
	}

	export := FightExport{
		Version:      1,
		InstanceID:   inst.InstanceID,
		GroupID:      inst.GroupID,
		EncounterID:  inst.EncounterID,
		StartTime:    inst.StartedAt,
		DurationMS:   int(inst.Duration.Milliseconds()),
		Outcome:      string(inst.Outcome),
		Source:       string(inst.Source),
		Participants: inst.Participants,
		Events:       convertEvents(events),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(export)
}

// getReplay returns replay data: instance metadata + combat events + WorldState frames.
func (h *Handler) getReplay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing instance id", http.StatusBadRequest)
		return
	}

	inst, err := h.repo.GetInstance(r.Context(), id)
	if err != nil {
		if errors.Is(err, combatlog.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events, err := h.repo.GetEvents(r.Context(), id, combatlog.EventFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []combatlog.LogEntry{}
	}

	frames, err := h.repo.GetReplay(r.Context(), id)
	if err != nil {
		if errors.Is(err, combatlog.ErrNotFound) {
			http.Error(w, "no replay data", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode each frame as base64 for JSON transport.
	b64Frames := make([]string, len(frames))
	for i, f := range frames {
		b64Frames[i] = base64.StdEncoding.EncodeToString(f)
	}

	export := ReplayExport{
		Version:      1,
		InstanceID:   inst.InstanceID,
		EncounterID:  inst.EncounterID,
		ZoneID:       inst.ZoneID,
		TickRate:     20,
		FrameCount:   len(frames),
		DurationMS:   int(inst.Duration.Milliseconds()),
		Outcome:      string(inst.Outcome),
		Participants: inst.Participants,
		Events:       convertEvents(events),
		Frames:       b64Frames,
	}

	writeJSON(w, export)
}

// getEncounterStats returns aggregate combat stats across all instances of an encounter.
func (h *Handler) getEncounterStats(w http.ResponseWriter, r *http.Request) {
	encounterID := r.PathValue("encounter_id")
	if encounterID == "" {
		http.Error(w, "missing encounter_id", http.StatusBadRequest)
		return
	}

	filter := combatlog.InstanceFilter{
		EncounterID: encounterID,
		Source:      r.URL.Query().Get("source"),
	}

	stats, err := h.repo.GetEncounterStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	abilities := make([]BossAbilityDTO, len(stats.BossAbilities))
	for i, ab := range stats.BossAbilities {
		abilities[i] = BossAbilityDTO{
			AbilityID:   ab.AbilityID,
			TotalDamage: ab.TotalDamage,
			Hits:        ab.Hits,
			Kills:       ab.Kills,
			Dodges:      ab.Dodges,
		}
	}

	writeJSON(w, EncounterStatsResponse{
		InstanceDamage:  stats.InstanceDamage,
		InstanceHealing: stats.InstanceHealing,
		InstanceDeaths:  stats.InstanceDeaths,
		InstancePhases:  stats.InstancePhases,
		BossAbilities:   abilities,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
