package system

import (
	"math"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

const telegraphTickHz = 20.0

// ticksFromSecs converts seconds to whole ticks (20 Hz), rounded.
func ticksFromSecs(s float32) uint32 {
	if s <= 0 {
		return 0
	}
	return uint32(s*telegraphTickHz + 0.5)
}

// isTelegraphState reports whether the enemy is in an ability commit (telegraph)
// state, the only states for which we emit a telegraph.
func isTelegraphState(s entity.EnemyState) bool {
	switch s {
	case entity.EnemyMeleeTelegraph, entity.EnemyRangedTelegraph,
		entity.EnemyAoETelegraph, entity.EnemyChargeTelegraph:
		return true
	}
	return false
}

// buildTelegraphs produces the active telegraph descriptors for this tick: one
// per enemy currently committing an ability. Geometry and timing come from the
// phase-resolved ability definition (never the raw def — phase overrides scale
// the radius), so the client renders exactly the lethal zone the server will
// resolve.
func buildTelegraphs(w *World) []codec.TelegraphDesc {
	var out []codec.TelegraphDesc
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || !isTelegraphState(e.State) {
			continue
		}
		def := enemyai.DefRegistry[e.DefName]
		if def == nil {
			continue
		}
		abil := def.AbilityByIndex(e.ActiveAbility)
		if abil == nil {
			continue
		}
		resolved := def.ResolveAbility(abil, e.Phase)

		// Stable absolute tick window. StateTimer is the remaining commit time;
		// elapsed = CommitTime - StateTimer. Clamped so start never underflows.
		commitTicks := ticksFromSecs(resolved.CommitTime)
		elapsedTicks := min(ticksFromSecs(resolved.CommitTime-e.StateTimer), w.TickNum)
		start := w.TickNum - elapsedTicks
		exec := start + commitTicks

		d := codec.TelegraphDesc{
			ID:          uint32(e.ID),
			Category:    telegraphCategory(&resolved),
			StartTick:   start,
			ExecuteTick: exec,
		}
		if !fillTelegraphGeometry(&d, &resolved, e, w) {
			continue // no renderable geometry for this ability
		}
		out = append(out, d)
	}
	return out
}

// fillTelegraphGeometry sets the shape + geometry on d from the resolved ability.
// Returns false if the ability has no ground telegraph to draw.
func fillTelegraphGeometry(d *codec.TelegraphDesc, a *ability.AbilityDef, e *entity.Enemy, w *World) bool {
	switch {
	case a.Category == ability.CategoryCharge && a.Charge != nil:
		d.Shape = codec.TelegraphShapeLine
		d.CX, d.CZ = e.Position.X, e.Position.Z
		d.DirX, d.DirZ = e.ChargeDirection.X, e.ChargeDirection.Z
		d.Length = a.Charge.MaxDistance
		d.Width = 2 * a.Charge.HitRadius
		return true

	case a.Category == ability.CategoryRanged:
		dx := e.RangedTargetPos.X - e.Position.X
		dz := e.RangedTargetPos.Z - e.Position.Z
		dist := float32(math.Hypot(float64(dx), float64(dz)))
		if dist < 0.1 {
			return false
		}
		d.Shape = codec.TelegraphShapeLine
		d.CX, d.CZ = e.Position.X, e.Position.Z
		d.DirX, d.DirZ = dx/dist, dz/dist
		d.Length = dist
		d.Width = 0.5
		return true
	}

	switch a.Hit.Type {
	case ability.HitMeleeArc, ability.HitAoECone:
		d.Shape = codec.TelegraphShapeCone
		d.CX, d.CZ = e.Position.X, e.Position.Z
		d.Facing = e.RotationY
		d.HalfAngle = a.Hit.ArcDegrees * (math.Pi / 180.0) / 2.0
		d.Range = a.Hit.Range
		return true

	case ability.HitAoECircle:
		d.Shape = codec.TelegraphShapeCircle
		d.CX, d.CZ = e.Position.X, e.Position.Z
		d.Radius = a.Hit.Radius
		return true

	case ability.HitAoEObstacles:
		return fillObstacleTelegraph(d, a, w)
	}
	return false
}

// fillObstacleTelegraph builds the multi-ring telegraph for pillar_overload: one
// ring per pillar-like obstacle, sized to cover each pillar's lethal zone.
func fillObstacleTelegraph(d *codec.TelegraphDesc, a *ability.AbilityDef, w *World) bool {
	d.Shape = codec.TelegraphShapeMulti
	for i := range w.Obstacles {
		o := w.Obstacles[i]
		if !combat.IsPillarLike(o) {
			continue
		}
		// Damage extends `Radius` beyond the pillar's AABB edge; a ring of
		// Radius + the larger half-extent fully covers the lethal zone.
		ext := o.HX
		if o.HZ > ext {
			ext = o.HZ
		}
		if a.Hit.Radius+ext > d.Radius {
			d.Radius = a.Hit.Radius + ext
		}
		d.Centers = append(d.Centers, [2]float32{o.CX, o.CZ})
	}
	return len(d.Centers) > 0
}

// telegraphCategory resolves an ability's telegraph category: the explicit YAML
// override when set, otherwise the default derived from its damage source.
func telegraphCategory(a *ability.AbilityDef) uint8 {
	switch a.TelegraphCategory {
	case "parryable":
		return codec.TelegraphCatParryable
	case "blockable":
		return codec.TelegraphCatBlockable
	case "unavoidable":
		return codec.TelegraphCatUnavoidable
	case "heal":
		return codec.TelegraphCatHeal
	}
	switch a.DamageSource {
	case combat.SourceEnemyMelee:
		return codec.TelegraphCatParryable
	case combat.SourcePlayerHeal:
		return codec.TelegraphCatHeal
	default: // ranged / aoe / charge / pillar
		return codec.TelegraphCatUnavoidable
	}
}
