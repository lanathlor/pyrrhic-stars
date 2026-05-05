import type { LogEntry, ParticipantLog, DamageBreakdown, AbilityBreakdown, DamageTakenBreakdown, SourceBreakdown } from "../types";
import { EVENT_TYPES } from "../constants";

export function computeDamageDone(
  events: LogEntry[],
  participants: ParticipantLog[],
  durationMs: number
): DamageBreakdown[] {
  const durationSec = Math.max(durationMs / 1000, 1);
  const participantMap = new Map(participants.map((p) => [p.entity_id, p]));

  const bySource = new Map<string, { total: number; crits: number; hits: number; abilities: Map<string, { total: number; crits: number; hits: number; max: number; school: string }> }>();

  for (const e of events) {
    if (e.event_type !== EVENT_TYPES.DAMAGE) continue;
    if (!e.source.startsWith("player")) continue;

    let entry = bySource.get(e.source);
    if (!entry) {
      entry = { total: 0, crits: 0, hits: 0, abilities: new Map() };
      bySource.set(e.source, entry);
    }

    entry.total += e.amount;
    entry.hits++;
    if (e.is_crit) entry.crits++;

    const abilityKey = e.ability_id || "auto_attack";
    let ability = entry.abilities.get(abilityKey);
    if (!ability) {
      ability = { total: 0, crits: 0, hits: 0, max: 0, school: e.school };
      entry.abilities.set(abilityKey, ability);
    }
    ability.total += e.amount;
    ability.hits++;
    if (e.is_crit) ability.crits++;
    if (e.amount > ability.max) ability.max = e.amount;
  }

  const results: DamageBreakdown[] = [];
  for (const [entityId, data] of bySource) {
    const p = participantMap.get(entityId);
    const abilities: AbilityBreakdown[] = Array.from(data.abilities.entries())
      .map(([abilityId, a]) => ({
        abilityId,
        totalDamage: a.total,
        hitCount: a.hits,
        critCount: a.crits,
        critRate: a.hits > 0 ? a.crits / a.hits : 0,
        avgHit: a.hits > 0 ? a.total / a.hits : 0,
        maxHit: a.max,
        school: a.school,
      }))
      .sort((a, b) => b.totalDamage - a.totalDamage);

    results.push({
      entityId,
      name: p?.name ?? entityId,
      className: p?.class ?? "",
      totalDamage: data.total,
      dps: data.total / durationSec,
      abilities,
      critRate: data.hits > 0 ? data.crits / data.hits : 0,
      hitCount: data.hits,
    });
  }

  return results.sort((a, b) => b.totalDamage - a.totalDamage);
}

export function computeDamageTaken(
  events: LogEntry[],
  participants: ParticipantLog[],
  durationMs: number
): DamageTakenBreakdown[] {
  const durationSec = Math.max(durationMs / 1000, 1);
  const participantMap = new Map(participants.map((p) => [p.entity_id, p]));

  const byTarget = new Map<string, { total: number; sources: Map<string, { total: number; hits: number }> }>();

  for (const e of events) {
    if (e.event_type !== EVENT_TYPES.DAMAGE) continue;
    if (!e.target.startsWith("player")) continue;

    let entry = byTarget.get(e.target);
    if (!entry) {
      entry = { total: 0, sources: new Map() };
      byTarget.set(e.target, entry);
    }

    entry.total += e.amount;

    const sourceKey = `${e.source}|${e.ability_id || "auto_attack"}`;
    let source = entry.sources.get(sourceKey);
    if (!source) {
      source = { total: 0, hits: 0 };
      entry.sources.set(sourceKey, source);
    }
    source.total += e.amount;
    source.hits++;
  }

  const results: DamageTakenBreakdown[] = [];
  for (const [entityId, data] of byTarget) {
    const p = participantMap.get(entityId);
    const sources: SourceBreakdown[] = Array.from(data.sources.entries())
      .map(([key, s]) => {
        const [source, abilityId] = key.split("|");
        return { source, abilityId, totalDamage: s.total, hitCount: s.hits };
      })
      .sort((a, b) => b.totalDamage - a.totalDamage);

    results.push({
      entityId,
      name: p?.name ?? entityId,
      className: p?.class ?? "",
      totalDamageTaken: data.total,
      dtps: data.total / durationSec,
      sources,
    });
  }

  return results.sort((a, b) => b.totalDamageTaken - a.totalDamageTaken);
}
