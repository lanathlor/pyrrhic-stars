import type { LogEntry, ParticipantLog, FightSummary } from "../types";
import { EVENT_TYPES } from "../constants";

export function computeSummaryKPIs(
  events: LogEntry[],
  participants: ParticipantLog[],
  durationMs: number
): FightSummary {
  let totalDamage = 0;
  let totalHealing = 0;
  let deathCount = 0;
  const phases = new Set<string>();
  const durationSec = Math.max(durationMs / 1000, 1);

  for (const e of events) {
    if (e.phase) phases.add(e.phase);
    switch (e.event_type) {
      case EVENT_TYPES.DAMAGE:
        if (e.source.startsWith("player")) {
          totalDamage += e.amount;
        }
        break;
      case EVENT_TYPES.HEAL:
        totalHealing += e.amount;
        break;
      case EVENT_TYPES.DEATH:
        if (e.target.startsWith("player") || e.source.startsWith("player")) {
          deathCount++;
        }
        break;
    }
  }

  const playerCount = participants.filter((p) => p.entity_id.startsWith("player")).length;

  return {
    totalDamage,
    raidDps: totalDamage / durationSec,
    totalHealing,
    raidHps: totalHealing / durationSec,
    deathCount,
    fightDurationMs: durationMs,
    phases: Array.from(phases),
    playerCount,
  };
}
