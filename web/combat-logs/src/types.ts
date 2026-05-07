export interface InstanceLog {
  instance_id: string;
  group_id: string;
  encounter_id: string;
  started_at: string;
  duration_ms: number;
  outcome: string;
  source: string;
  participants: ParticipantLog[] | null;
}

export interface ParticipantLog {
  instance_id: string;
  entity_id: string;
  name: string;
  class: string;
  is_bot: boolean;
  bot_profile?: string;
}

export interface LogEntry {
  group_id: string;
  instance_id: string;
  encounter_id: string;
  tick: number;
  timestamp_ms: number;
  source: string;
  source_class: string;
  target: string;
  event_type: number;
  ability_id: string;
  amount: number;
  overkill: number;
  school: string;
  is_crit: boolean;
  is_dodged: boolean;
  phase: string;
  boss_health: number;
  pos_x: number;
  pos_y: number;
  pos_z: number;
  resource_type: string;
  resource_delta: number;
  resource_after: number;
}

export const EVENT_TYPE_NAMES: Record<number, string> = {
  1: "Damage",
  2: "Heal",
  3: "Buff Apply",
  4: "Buff Remove",
  5: "Buff Tick",
  6: "Cast Start",
  7: "Cast End",
  8: "CD Start",
  9: "CD End",
  10: "Dodge",
  11: "Death",
  12: "Phase Change",
};

// --- Computed analysis types ---

export interface AbilityBreakdown {
  abilityId: string;
  totalDamage: number;
  hitCount: number;
  critCount: number;
  critRate: number;
  avgHit: number;
  maxHit: number;
  school: string;
}

export interface DamageBreakdown {
  entityId: string;
  name: string;
  className: string;
  totalDamage: number;
  dps: number;
  abilities: AbilityBreakdown[];
  critRate: number;
  hitCount: number;
}

export interface HealingBreakdown {
  entityId: string;
  name: string;
  className: string;
  totalHealing: number;
  hps: number;
  abilities: AbilityBreakdown[];
  critRate: number;
  hitCount: number;
}

export interface DamageTakenBreakdown {
  entityId: string;
  name: string;
  className: string;
  totalDamageTaken: number;
  dtps: number;
  sources: SourceBreakdown[];
}

export interface SourceBreakdown {
  source: string;
  abilityId: string;
  totalDamage: number;
  hitCount: number;
}

export interface DeathReport {
  tick: number;
  timestampMs: number;
  victim: string;
  victimName: string;
  victimClass: string;
  killingBlow: LogEntry | null;
  leadup: LogEntry[];
}

export interface TimelinePoint {
  timestampMs: number;
  value: number;
  label?: string;
}

export interface DPSTimelinePoint {
  timestampMs: number;
  [entityId: string]: number;
}

export interface PhaseMarker {
  phase: string;
  startMs: number;
  endMs: number;
}

export interface FightSummary {
  totalDamage: number;
  raidDps: number;
  totalHealing: number;
  raidHps: number;
  deathCount: number;
  fightDurationMs: number;
  phases: string[];
  playerCount: number;
}

// --- Aggregate report types (across many runs) ---

export interface EncounterSummary {
  encounterId: string;
  totalRuns: number;
  wins: number;
  losses: number;
  timeouts: number;
  winRate: number;
  avgDurationMs: number;
  firstRun: string; // ISO date
  lastRun: string;
  profiles: string[]; // distinct bot profiles seen
  classes: string[]; // distinct classes seen
}

export interface ReportOverview {
  encounterId: string;
  totalRuns: number;
  wins: number;
  losses: number;
  timeouts: number;
  winRate: number;
  durationStats: PercentileStats;
  firstRun: string;
  lastRun: string;
}

export interface PercentileStats {
  min: number;
  max: number;
  avg: number;
  median: number;
  p95: number;
  p99: number;
  values: number[]; // sorted, for histogram
}

export interface ProfileStats {
  profile: string;
  runs: number;
  wins: number;
  losses: number;
  timeouts: number;
  winRate: number;
  avgDurationMs: number;
  durationStats: PercentileStats;
}

export interface CompStats {
  name: string; // e.g. "gunner + vanguard (sweaty)"
  classes: string[];
  profiles: string[];
  runs: number;
  wins: number;
  losses: number;
  timeouts: number;
  winRate: number;
  avgDurationMs: number;
  durationStats: PercentileStats;
}

export interface DurationBucket {
  rangeLabel: string;
  minMs: number;
  maxMs: number;
  count: number;
}

// --- Encounter aggregate stats (from server) ---

export interface EncounterStats {
  instance_damage: Record<string, Record<string, number>>;  // instance_id → class → total_damage
  instance_healing: Record<string, Record<string, number>>; // instance_id → class → total_healing
  instance_deaths: Record<string, number>;                   // instance_id → death_count
  instance_phases: Record<string, string>;                   // instance_id → max_phase
  boss_abilities: BossAbilityStat[];
}

export interface BossAbilityStat {
  ability_id: string;
  total_damage: number;
  hits: number;
  kills: number;
  dodges: number;
}

// --- Computed aggregate types ---

export interface ClassDPSDistribution {
  className: string;
  stats: PercentileStats;
}

export interface ProfileCombatStats extends ProfileStats {
  classDPS: ClassDPSDistribution[];
  deathStats: PercentileStats;
  phaseReach: Record<string, number>; // phase → reach rate (0-1)
}

export interface PhaseReachEntry {
  phase: string;
  rate: number;  // 0-1
  count: number;
}

export interface WipePhaseEntry {
  phase: string;
  count: number;
}
