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
