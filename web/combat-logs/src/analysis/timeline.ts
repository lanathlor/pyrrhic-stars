import type { LogEntry, ParticipantLog, TimelinePoint, DPSTimelinePoint, PhaseMarker } from "../types";
import { EVENT_TYPES } from "../constants";

const MAX_CHART_POINTS = 300;

export function computeBossHPTimeline(events: LogEntry[]): TimelinePoint[] {
  const points: TimelinePoint[] = [];
  let lastHP = -1;

  for (const e of events) {
    if (e.boss_health > 0 && e.boss_health !== lastHP) {
      points.push({ timestampMs: e.timestamp_ms, value: e.boss_health * 100 });
      lastHP = e.boss_health;
    }
  }

  // Downsample if too many points
  if (points.length > MAX_CHART_POINTS) {
    return downsampleTimeline(points, MAX_CHART_POINTS);
  }
  return points;
}

export function computeDPSTimeline(
  events: LogEntry[],
  participants: ParticipantLog[],
  durationMs: number,
  windowMs: number = 5000
): DPSTimelinePoint[] {
  if (durationMs <= 0) return [];

  const playerIds = participants
    .filter((p) => p.entity_id.startsWith("player"))
    .map((p) => p.entity_id);

  if (playerIds.length === 0) return [];

  // Adaptive bucket size: at most MAX_CHART_POINTS output points
  const durationSec = durationMs / 1000;
  const bucketSizeSec = Math.max(1, Math.ceil(durationSec / MAX_CHART_POINTS));
  const bucketCount = Math.min(Math.ceil(durationSec / bucketSizeSec), MAX_CHART_POINTS);

  if (bucketCount <= 0) return [];

  const buckets = new Map<string, Float64Array>();
  for (const pid of playerIds) {
    buckets.set(pid, new Float64Array(bucketCount));
  }

  for (const e of events) {
    if (e.event_type !== EVENT_TYPES.DAMAGE) continue;
    if (!e.source.startsWith("player")) continue;
    const bucket = buckets.get(e.source);
    if (!bucket) continue;
    const idx = Math.min(Math.floor(e.timestamp_ms / 1000 / bucketSizeSec), bucketCount - 1);
    bucket[idx] += e.amount;
  }

  // Sliding window DPS (window size in buckets)
  const halfWindow = Math.max(1, Math.floor(windowMs / 1000 / bucketSizeSec / 2));
  const points: DPSTimelinePoint[] = [];

  for (let b = 0; b < bucketCount; b++) {
    const point: DPSTimelinePoint = { timestampMs: b * bucketSizeSec * 1000 };
    const lo = Math.max(0, b - halfWindow);
    const hi = Math.min(bucketCount - 1, b + halfWindow);
    const windowSec = (hi - lo + 1) * bucketSizeSec;

    for (const pid of playerIds) {
      const bucket = buckets.get(pid)!;
      let sum = 0;
      for (let i = lo; i <= hi; i++) sum += bucket[i];
      point[pid] = sum / windowSec;
    }
    points.push(point);
  }

  return points;
}

export function computePhaseMarkers(events: LogEntry[], durationMs: number): PhaseMarker[] {
  const phases: PhaseMarker[] = [];
  let currentPhase = "";
  let phaseStart = 0;

  for (const e of events) {
    if (e.phase && e.phase !== currentPhase) {
      if (currentPhase) {
        phases.push({ phase: currentPhase, startMs: phaseStart, endMs: e.timestamp_ms });
      }
      currentPhase = e.phase;
      phaseStart = e.timestamp_ms;
    }
  }

  if (currentPhase) {
    phases.push({ phase: currentPhase, startMs: phaseStart, endMs: durationMs });
  }

  return phases;
}

function downsampleTimeline(points: TimelinePoint[], maxPoints: number): TimelinePoint[] {
  if (points.length <= maxPoints) return points;
  const step = points.length / maxPoints;
  const result: TimelinePoint[] = [];
  for (let i = 0; i < maxPoints; i++) {
    result.push(points[Math.floor(i * step)]);
  }
  // Always include the last point
  if (result[result.length - 1] !== points[points.length - 1]) {
    result.push(points[points.length - 1]);
  }
  return result;
}
