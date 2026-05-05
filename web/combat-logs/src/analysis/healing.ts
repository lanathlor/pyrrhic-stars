import type { LogEntry, ParticipantLog, HealingBreakdown, AbilityBreakdown } from "../types";
import { EVENT_TYPES } from "../constants";

export function computeHealingDone(
  events: LogEntry[],
  participants: ParticipantLog[],
  durationMs: number
): HealingBreakdown[] {
  const durationSec = Math.max(durationMs / 1000, 1);
  const participantMap = new Map(participants.map((p) => [p.entity_id, p]));

  const bySource = new Map<string, { total: number; crits: number; hits: number; abilities: Map<string, { total: number; crits: number; hits: number; max: number; school: string }> }>();

  for (const e of events) {
    if (e.event_type !== EVENT_TYPES.HEAL) continue;

    let entry = bySource.get(e.source);
    if (!entry) {
      entry = { total: 0, crits: 0, hits: 0, abilities: new Map() };
      bySource.set(e.source, entry);
    }

    entry.total += e.amount;
    entry.hits++;
    if (e.is_crit) entry.crits++;

    const abilityKey = e.ability_id || "auto_heal";
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

  const results: HealingBreakdown[] = [];
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
      totalHealing: data.total,
      hps: data.total / durationSec,
      abilities,
      critRate: data.hits > 0 ? data.crits / data.hits : 0,
      hitCount: data.hits,
    });
  }

  return results.sort((a, b) => b.totalHealing - a.totalHealing);
}
