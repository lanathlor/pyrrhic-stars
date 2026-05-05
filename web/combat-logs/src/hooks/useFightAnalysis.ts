import { useMemo } from "react";
import type { InstanceLog, LogEntry } from "../types";
import {
  filterByPhase,
  computeSummaryKPIs,
  computeDamageDone,
  computeDamageTaken,
  computeHealingDone,
  computeDeathReports,
  computeBossHPTimeline,
  computeDPSTimeline,
  computePhaseMarkers,
} from "../analysis";

/**
 * Normalize event timestamps to be fight-relative (start at 0) and compute
 * the effective fight duration from the event stream. This handles the case
 * where the server stores absolute tick offsets instead of fight-relative ones.
 */
function normalizeEvents(events: LogEntry[]): { normalized: LogEntry[]; durationMs: number } {
  if (events.length === 0) return { normalized: [], durationMs: 0 };
  const minTs = events[0].timestamp_ms; // events are sorted by tick
  if (minTs === 0) {
    // Already fight-relative
    const maxTs = events[events.length - 1].timestamp_ms;
    return { normalized: events, durationMs: Math.max(maxTs, 1) };
  }
  // Shift all timestamps so the fight starts at 0
  const normalized = events.map((e) => ({ ...e, timestamp_ms: e.timestamp_ms - minTs }));
  const maxTs = normalized[normalized.length - 1].timestamp_ms;
  return { normalized, durationMs: Math.max(maxTs, 1) };
}

export function useFightAnalysis(
  instance: InstanceLog | null,
  events: LogEntry[],
  selectedPhase: string | null
) {
  return useMemo(() => {
    if (!instance) {
      return {
        filteredEvents: [],
        effectiveDurationMs: 0,
        summary: { totalDamage: 0, raidDps: 0, totalHealing: 0, raidHps: 0, deathCount: 0, fightDurationMs: 0, phases: [], playerCount: 0 },
        damageDone: [],
        damageTaken: [],
        healingDone: [],
        deaths: [],
        bossHP: [],
        dpsTimeline: [],
        phases: [],
      };
    }

    const participants = instance.participants ?? [];
    const { normalized, durationMs } = normalizeEvents(events);
    const filtered = filterByPhase(normalized, selectedPhase);

    // When filtering by phase, use phase duration for DPS calculations
    let effectiveDuration = durationMs;
    if (selectedPhase && filtered.length > 0) {
      const first = filtered[0].timestamp_ms;
      const last = filtered[filtered.length - 1].timestamp_ms;
      effectiveDuration = Math.max(last - first, 1);
    }

    const summary = computeSummaryKPIs(filtered, participants, effectiveDuration);
    const damageDone = computeDamageDone(filtered, participants, effectiveDuration);
    const damageTaken = computeDamageTaken(filtered, participants, effectiveDuration);
    const healingDone = computeHealingDone(filtered, participants, effectiveDuration);
    const deaths = computeDeathReports(filtered, participants);
    // Timeline always uses full normalized event set for context
    const bossHP = computeBossHPTimeline(normalized);
    const dpsTimeline = computeDPSTimeline(normalized, participants, durationMs);
    const phases = computePhaseMarkers(normalized, durationMs);

    return {
      filteredEvents: filtered,
      effectiveDurationMs: effectiveDuration,
      summary,
      damageDone,
      damageTaken,
      healingDone,
      deaths,
      bossHP,
      dpsTimeline,
      phases,
    };
  }, [instance, events, selectedPhase]);
}
