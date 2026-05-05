import type { LogEntry, ParticipantLog, DeathReport } from "../types";
import { EVENT_TYPES } from "../constants";

export function computeDeathReports(
  events: LogEntry[],
  participants: ParticipantLog[]
): DeathReport[] {
  const participantMap = new Map(participants.map((p) => [p.entity_id, p]));
  const reports: DeathReport[] = [];

  for (let i = 0; i < events.length; i++) {
    const e = events[i];
    if (e.event_type !== EVENT_TYPES.DEATH) continue;

    // The death event target is the one who died; if not set, check source
    const victim = e.target || e.source;
    if (!victim.startsWith("player")) continue;

    const p = participantMap.get(victim);

    // Walk backwards to find the last 10 events targeting this player
    const leadup: LogEntry[] = [];
    let killingBlow: LogEntry | null = null;

    for (let j = i - 1; j >= 0 && leadup.length < 10; j--) {
      const prev = events[j];
      if (prev.target === victim || prev.source === victim) {
        leadup.unshift(prev);
        if (
          !killingBlow &&
          prev.event_type === EVENT_TYPES.DAMAGE &&
          prev.target === victim
        ) {
          killingBlow = prev;
        }
      }
    }

    reports.push({
      tick: e.tick,
      timestampMs: e.timestamp_ms,
      victim,
      victimName: p?.name ?? victim,
      victimClass: p?.class ?? "",
      killingBlow,
      leadup,
    });
  }

  return reports.sort((a, b) => a.timestampMs - b.timestampMs);
}
