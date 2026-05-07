import type {
  InstanceLog,
  EncounterSummary,
  ReportOverview,
  ProfileStats,
  CompStats,
  PercentileStats,
  DurationBucket,
  EncounterStats,
  ClassDPSDistribution,
  ProfileCombatStats,
  PhaseReachEntry,
  WipePhaseEntry,
} from "../types";

// ── Percentile helpers ──

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = (p / 100) * (sorted.length - 1);
  const lo = Math.floor(idx);
  const hi = Math.ceil(idx);
  if (lo === hi) return sorted[lo];
  return sorted[lo] + (sorted[hi] - sorted[lo]) * (idx - lo);
}

function computePercentileStats(values: number[]): PercentileStats {
  if (values.length === 0) {
    return { min: 0, max: 0, avg: 0, median: 0, p95: 0, p99: 0, values: [] };
  }
  const sorted = [...values].sort((a, b) => a - b);
  const sum = sorted.reduce((s, v) => s + v, 0);
  return {
    min: sorted[0],
    max: sorted[sorted.length - 1],
    avg: sum / sorted.length,
    median: percentile(sorted, 50),
    p95: percentile(sorted, 95),
    p99: percentile(sorted, 99),
    values: sorted,
  };
}

// ── Encounter grouping (landing page) ──

export function groupByEncounter(instances: InstanceLog[]): EncounterSummary[] {
  const map = new Map<string, InstanceLog[]>();
  for (const inst of instances) {
    const list = map.get(inst.encounter_id) ?? [];
    list.push(inst);
    map.set(inst.encounter_id, list);
  }

  const summaries: EncounterSummary[] = [];
  for (const [encounterId, runs] of map) {
    let wins = 0, losses = 0, timeouts = 0;
    let totalDur = 0;
    const profiles = new Set<string>();
    const classes = new Set<string>();
    let firstRun = runs[0].started_at;
    let lastRun = runs[0].started_at;

    for (const r of runs) {
      if (r.outcome === "player_win") wins++;
      else if (r.outcome === "boss_win") losses++;
      else timeouts++;
      totalDur += r.duration_ms;
      if (r.started_at < firstRun) firstRun = r.started_at;
      if (r.started_at > lastRun) lastRun = r.started_at;
      for (const p of r.participants ?? []) {
        if (p.entity_id.startsWith("player")) {
          if (p.bot_profile) profiles.add(p.bot_profile);
          if (p.class) classes.add(p.class);
        }
      }
    }

    summaries.push({
      encounterId,
      totalRuns: runs.length,
      wins,
      losses,
      timeouts,
      winRate: runs.length > 0 ? wins / runs.length : 0,
      avgDurationMs: runs.length > 0 ? totalDur / runs.length : 0,
      firstRun,
      lastRun,
      profiles: Array.from(profiles).sort(),
      classes: Array.from(classes).sort(),
    });
  }

  return summaries.sort((a, b) => b.totalRuns - a.totalRuns);
}

// ── Report overview ──

export function computeReportOverview(instances: InstanceLog[]): ReportOverview {
  let wins = 0, losses = 0, timeouts = 0;
  let firstRun = instances[0]?.started_at ?? "";
  let lastRun = instances[0]?.started_at ?? "";
  const durations: number[] = [];

  for (const inst of instances) {
    if (inst.outcome === "player_win") wins++;
    else if (inst.outcome === "boss_win") losses++;
    else timeouts++;
    durations.push(inst.duration_ms);
    if (inst.started_at < firstRun) firstRun = inst.started_at;
    if (inst.started_at > lastRun) lastRun = inst.started_at;
  }

  return {
    encounterId: instances[0]?.encounter_id ?? "",
    totalRuns: instances.length,
    wins,
    losses,
    timeouts,
    winRate: instances.length > 0 ? wins / instances.length : 0,
    durationStats: computePercentileStats(durations),
    firstRun,
    lastRun,
  };
}

// ── Per-profile breakdown ──

export function computeProfileStats(instances: InstanceLog[]): ProfileStats[] {
  // Group runs by the dominant profile (most common among participants)
  const map = new Map<string, InstanceLog[]>();
  for (const inst of instances) {
    const profile = dominantProfile(inst);
    if (!profile) continue;
    const list = map.get(profile) ?? [];
    list.push(inst);
    map.set(profile, list);
  }

  const stats: ProfileStats[] = [];
  for (const [profile, runs] of map) {
    let wins = 0, losses = 0, timeouts = 0;
    const durations: number[] = [];
    for (const r of runs) {
      if (r.outcome === "player_win") wins++;
      else if (r.outcome === "boss_win") losses++;
      else timeouts++;
      durations.push(r.duration_ms);
    }
    stats.push({
      profile,
      runs: runs.length,
      wins,
      losses,
      timeouts,
      winRate: runs.length > 0 ? wins / runs.length : 0,
      avgDurationMs: runs.length > 0 ? durations.reduce((a, b) => a + b, 0) / runs.length : 0,
      durationStats: computePercentileStats(durations),
    });
  }

  // Order: sweaty, average, bad, then alphabetical
  const order: Record<string, number> = { sweaty: 0, average: 1, bad: 2 };
  return stats.sort((a, b) => (order[a.profile] ?? 99) - (order[b.profile] ?? 99));
}

function dominantProfile(inst: InstanceLog): string | null {
  const players = (inst.participants ?? []).filter((p) => p.entity_id.startsWith("player"));
  if (players.length === 0) return null;
  const counts = new Map<string, number>();
  for (const p of players) {
    const prof = p.bot_profile || "live";
    counts.set(prof, (counts.get(prof) ?? 0) + 1);
  }
  let best = "";
  let bestCount = 0;
  for (const [prof, count] of counts) {
    if (count > bestCount) {
      best = prof;
      bestCount = count;
    }
  }
  return best;
}

// ── Per-composition breakdown ──

export function computeCompStats(instances: InstanceLog[]): CompStats[] {
  const map = new Map<string, { classes: string[]; profiles: string[]; runs: InstanceLog[] }>();

  for (const inst of instances) {
    const players = (inst.participants ?? []).filter((p) => p.entity_id.startsWith("player"));
    const classes = players.map((p) => p.class).sort();
    const profiles = players.map((p) => p.bot_profile || "live").sort();
    const key = classes.join("+") + " (" + profiles.join("+") + ")";
    const existing = map.get(key);
    if (existing) {
      existing.runs.push(inst);
    } else {
      map.set(key, { classes, profiles, runs: [inst] });
    }
  }

  const stats: CompStats[] = [];
  for (const [name, { classes, profiles, runs }] of map) {
    let wins = 0, losses = 0, timeouts = 0;
    const durations: number[] = [];
    for (const r of runs) {
      if (r.outcome === "player_win") wins++;
      else if (r.outcome === "boss_win") losses++;
      else timeouts++;
      durations.push(r.duration_ms);
    }
    stats.push({
      name,
      classes,
      profiles,
      runs: runs.length,
      wins,
      losses,
      timeouts,
      winRate: runs.length > 0 ? wins / runs.length : 0,
      avgDurationMs: runs.length > 0 ? durations.reduce((a, b) => a + b, 0) / runs.length : 0,
      durationStats: computePercentileStats(durations),
    });
  }

  return stats.sort((a, b) => b.winRate - a.winRate);
}

// ── Duration histogram ──

export function computeDurationHistogram(
  values: number[],
  bucketCount = 20
): DurationBucket[] {
  if (values.length === 0) return [];
  const sorted = [...values].sort((a, b) => a - b);
  const min = sorted[0];
  const max = sorted[sorted.length - 1];
  const range = max - min;
  if (range === 0) {
    return [{ rangeLabel: formatBucketMs(min), minMs: min, maxMs: max, count: values.length }];
  }

  const step = range / bucketCount;
  const buckets: DurationBucket[] = [];
  for (let i = 0; i < bucketCount; i++) {
    const lo = min + i * step;
    const hi = i === bucketCount - 1 ? max + 1 : min + (i + 1) * step;
    buckets.push({
      rangeLabel: formatBucketMs(lo),
      minMs: lo,
      maxMs: hi,
      count: 0,
    });
  }

  for (const v of sorted) {
    const idx = Math.min(Math.floor((v - min) / step), bucketCount - 1);
    buckets[idx].count++;
  }

  return buckets;
}

function formatBucketMs(ms: number): string {
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(0)}s`;
  const m = Math.floor(s / 60);
  const rem = Math.round(s % 60);
  return `${m}:${rem.toString().padStart(2, "0")}`;
}

// ── Aggregate combat stats (from encounter stats endpoint) ──

/** Build a duration map for quick lookup: instance_id → duration_ms */
function buildDurationMap(instances: InstanceLog[]): Map<string, number> {
  const m = new Map<string, number>();
  for (const inst of instances) m.set(inst.instance_id, inst.duration_ms);
  return m;
}

/** Per-class DPS distributions across all runs */
export function computeClassDPSStats(
  stats: EncounterStats,
  instances: InstanceLog[]
): ClassDPSDistribution[] {
  const durMap = buildDurationMap(instances);
  const byClass = new Map<string, number[]>();

  for (const [instId, classDmg] of Object.entries(stats.instance_damage)) {
    const durMs = durMap.get(instId);
    if (!durMs || durMs <= 0) continue;
    const durSec = durMs / 1000;
    for (const [cls, dmg] of Object.entries(classDmg)) {
      const arr = byClass.get(cls) ?? [];
      arr.push(dmg / durSec);
      byClass.set(cls, arr);
    }
  }

  const result: ClassDPSDistribution[] = [];
  for (const [cls, values] of byClass) {
    result.push({ className: cls, stats: computePercentileStats(values) });
  }
  return result.sort((a, b) => b.stats.median - a.stats.median);
}

/** Per-class HPS distributions across all runs */
export function computeClassHPSStats(
  stats: EncounterStats,
  instances: InstanceLog[]
): ClassDPSDistribution[] {
  const durMap = buildDurationMap(instances);
  const byClass = new Map<string, number[]>();

  for (const [instId, classHeal] of Object.entries(stats.instance_healing)) {
    const durMs = durMap.get(instId);
    if (!durMs || durMs <= 0) continue;
    const durSec = durMs / 1000;
    for (const [cls, heal] of Object.entries(classHeal)) {
      const arr = byClass.get(cls) ?? [];
      arr.push(heal / durSec);
      byClass.set(cls, arr);
    }
  }

  const result: ClassDPSDistribution[] = [];
  for (const [cls, values] of byClass) {
    result.push({ className: cls, stats: computePercentileStats(values) });
  }
  return result.sort((a, b) => b.stats.median - a.stats.median);
}

/** Deaths-per-run distribution */
export function computeDeathStats(
  stats: EncounterStats,
  instances: InstanceLog[]
): PercentileStats {
  const values = instances.map(
    (inst) => stats.instance_deaths[inst.instance_id] ?? 0
  );
  return computePercentileStats(values);
}

/** Raid-wide DPS distribution (total party damage / duration per run) */
export function computeRaidDPSStats(
  stats: EncounterStats,
  instances: InstanceLog[]
): PercentileStats {
  const durMap = buildDurationMap(instances);
  const values: number[] = [];

  for (const inst of instances) {
    const classDmg = stats.instance_damage[inst.instance_id];
    if (!classDmg) continue;
    const durMs = durMap.get(inst.instance_id);
    if (!durMs || durMs <= 0) continue;
    const totalDmg = Object.values(classDmg).reduce((s, d) => s + d, 0);
    values.push(totalDmg / (durMs / 1000));
  }

  return computePercentileStats(values);
}

/** Phase reach rates — what % of runs reached each phase */
export function computePhaseReach(
  stats: EncounterStats,
  instances: InstanceLog[]
): PhaseReachEntry[] {
  const total = instances.length;
  if (total === 0) return [];

  // Count how many runs reached each phase.
  // Phases are like "phase_1", "phase_2". If a run reached phase_3, it also reached 1 and 2.
  const allPhases = new Set<string>();
  const maxPhasePerInstance = new Map<string, string>();

  for (const [instId, phase] of Object.entries(stats.instance_phases)) {
    allPhases.add(phase);
    maxPhasePerInstance.set(instId, phase);
  }

  // Sort phases naturally (phase_1 < phase_2 < phase_3)
  const sortedPhases = Array.from(allPhases).sort();

  // For each phase, count runs that reached at least that phase
  const phaseCounts = new Map<string, number>();
  for (const phase of sortedPhases) phaseCounts.set(phase, 0);

  for (const maxPhase of maxPhasePerInstance.values()) {
    for (const phase of sortedPhases) {
      if (phase <= maxPhase) {
        phaseCounts.set(phase, (phaseCounts.get(phase) ?? 0) + 1);
      }
    }
  }

  return sortedPhases.map((phase) => ({
    phase,
    rate: (phaseCounts.get(phase) ?? 0) / total,
    count: phaseCounts.get(phase) ?? 0,
  }));
}

/** Wipe distribution — which phase do losses happen in */
export function computeWipePhases(
  stats: EncounterStats,
  instances: InstanceLog[]
): WipePhaseEntry[] {
  const counts = new Map<string, number>();

  for (const inst of instances) {
    if (inst.outcome !== "boss_win") continue;
    const phase = stats.instance_phases[inst.instance_id] ?? "unknown";
    counts.set(phase, (counts.get(phase) ?? 0) + 1);
  }

  const entries: WipePhaseEntry[] = [];
  for (const [phase, count] of counts) {
    entries.push({ phase, count });
  }
  return entries.sort((a, b) => a.phase.localeCompare(b.phase));
}

/** Per-profile combat stats — extends ProfileStats with DPS, deaths, phase reach */
export function computeProfileCombatStats(
  stats: EncounterStats,
  instances: InstanceLog[]
): ProfileCombatStats[] {
  // Group instances by dominant profile (reuse existing logic)
  const profileGroups = new Map<string, InstanceLog[]>();
  for (const inst of instances) {
    const profile = dominantProfile(inst);
    if (!profile) continue;
    const list = profileGroups.get(profile) ?? [];
    list.push(inst);
    profileGroups.set(profile, list);
  }

  const result: ProfileCombatStats[] = [];

  for (const [profile, runs] of profileGroups) {
    // Base profile stats
    let wins = 0, losses = 0, timeouts = 0;
    const durations: number[] = [];
    for (const r of runs) {
      if (r.outcome === "player_win") wins++;
      else if (r.outcome === "boss_win") losses++;
      else timeouts++;
      durations.push(r.duration_ms);
    }

    // Subset encounter stats to this profile's instances
    const subStats: EncounterStats = {
      instance_damage: {},
      instance_healing: {},
      instance_deaths: {},
      instance_phases: {},
      boss_abilities: stats.boss_abilities,
    };
    for (const inst of runs) {
      const id = inst.instance_id;
      if (stats.instance_damage[id]) subStats.instance_damage[id] = stats.instance_damage[id];
      if (stats.instance_healing[id]) subStats.instance_healing[id] = stats.instance_healing[id];
      if (id in stats.instance_deaths) subStats.instance_deaths[id] = stats.instance_deaths[id];
      if (stats.instance_phases[id]) subStats.instance_phases[id] = stats.instance_phases[id];
    }

    const classDPS = computeClassDPSStats(subStats, runs);
    const deathStats = computeDeathStats(subStats, runs);
    const phaseReachEntries = computePhaseReach(subStats, runs);
    const phaseReach: Record<string, number> = {};
    for (const e of phaseReachEntries) phaseReach[e.phase] = e.rate;

    result.push({
      profile,
      runs: runs.length,
      wins,
      losses,
      timeouts,
      winRate: runs.length > 0 ? wins / runs.length : 0,
      avgDurationMs: runs.length > 0 ? durations.reduce((a, b) => a + b, 0) / runs.length : 0,
      durationStats: computePercentileStats(durations),
      classDPS,
      deathStats,
      phaseReach,
    });
  }

  const order: Record<string, number> = { sweaty: 0, average: 1, bad: 2 };
  return result.sort((a, b) => (order[a.profile] ?? 99) - (order[b.profile] ?? 99));
}
