package combat

import (
	"math/rand/v2"

	"codex-online/server/internal/entity"
)

// ActivePattern is the runtime state of a pattern being executed.
type ActivePattern struct {
	Handle    PatternHandle
	Def       *PatternDef
	OwnerID   uint16
	EnemyIdx  int
	Origin    entity.Vec3
	Facing    entity.Vec3
	VisualTag string

	// Emitter sequencing
	EmitterIdx int
	WaveIdx    int
	WaveTimer  float32
	FirstTick  bool // first wave fires immediately
	Done       bool
}

// PatternEngine manages all active patterns and produces spawn requests each tick.
type PatternEngine struct {
	nextHandle PatternHandle
	Active     []*ActivePattern
	Pending    []SpawnRequest
}

// NewPatternEngine creates an empty pattern engine.
func NewPatternEngine() *PatternEngine {
	return &PatternEngine{}
}

// Spawn creates a new active pattern and returns its handle.
func (pe *PatternEngine) Spawn(def *PatternDef, visualTag string, ownerID uint16,
	enemyIdx int, origin, facing entity.Vec3) PatternHandle {
	if def == nil || len(def.Emitters) == 0 {
		return 0
	}
	pe.nextHandle++
	ap := &ActivePattern{
		Handle:    pe.nextHandle,
		Def:       def,
		OwnerID:   ownerID,
		EnemyIdx:  enemyIdx,
		Origin:    origin,
		Facing:    facing,
		VisualTag: visualTag,
		FirstTick: true,
	}
	pe.Active = append(pe.Active, ap)
	return pe.nextHandle
}

// Tick advances all active patterns by dt seconds.
// Accumulated spawn requests are retrievable via DrainSpawns.
func (pe *PatternEngine) Tick(dt float32, rng *rand.Rand) {
	pe.Pending = pe.Pending[:0]

	alive := pe.Active[:0]
	for _, ap := range pe.Active {
		pe.tickPattern(ap, dt, rng)
		if !ap.Done {
			alive = append(alive, ap)
		}
	}
	pe.Active = alive
}

// DrainSpawns returns the pending spawn requests from the last Tick and clears them.
func (pe *PatternEngine) DrainSpawns() []SpawnRequest {
	return pe.Pending
}

// Cancel removes an active pattern by handle.
func (pe *PatternEngine) Cancel(h PatternHandle) {
	for i, ap := range pe.Active {
		if ap.Handle == h {
			pe.Active[i] = pe.Active[len(pe.Active)-1]
			pe.Active = pe.Active[:len(pe.Active)-1]
			return
		}
	}
}

// IsActive returns true if a pattern with the given handle is still running.
func (pe *PatternEngine) IsActive(h PatternHandle) bool {
	for _, ap := range pe.Active {
		if ap.Handle == h {
			return true
		}
	}
	return false
}

// ActiveCount returns the number of patterns currently executing.
func (pe *PatternEngine) ActiveCount() int {
	return len(pe.Active)
}

func (pe *PatternEngine) tickPattern(ap *ActivePattern, dt float32, rng *rand.Rand) {
	if ap.EmitterIdx >= len(ap.Def.Emitters) {
		ap.Done = true
		return
	}

	emitter := &ap.Def.Emitters[ap.EmitterIdx]
	waves := emitter.Waves
	if waves <= 0 {
		waves = 1
	}

	// First wave fires immediately on spawn tick
	if ap.FirstTick {
		ap.FirstTick = false
		pe.fireWave(ap, emitter, rng)
		ap.WaveIdx++
		if ap.WaveIdx >= waves {
			pe.advanceEmitter(ap)
		}
		return
	}

	// Accumulate time and fire waves when interval reached
	ap.WaveTimer += dt
	for ap.WaveIdx < waves && ap.WaveTimer >= emitter.WaveInterval {
		ap.WaveTimer -= emitter.WaveInterval
		pe.fireWave(ap, emitter, rng)
		ap.WaveIdx++
	}

	if ap.WaveIdx >= waves {
		pe.advanceEmitter(ap)
	}
}

func (pe *PatternEngine) advanceEmitter(ap *ActivePattern) {
	ap.EmitterIdx++
	ap.WaveIdx = 0
	ap.WaveTimer = 0
	ap.FirstTick = true
	if ap.EmitterIdx >= len(ap.Def.Emitters) {
		ap.Done = true
	}
}

func (pe *PatternEngine) fireWave(ap *ActivePattern, emitter *EmitterDef, rng *rand.Rand) {
	requests := emitWave(emitter, ap.Origin, ap.Facing, ap.WaveIdx, rng)
	for i := range requests {
		requests[i].OwnerID = ap.OwnerID
		requests[i].EnemyIdx = ap.EnemyIdx
		requests[i].VisualTag = ap.VisualTag
	}
	pe.Pending = append(pe.Pending, requests...)
}
