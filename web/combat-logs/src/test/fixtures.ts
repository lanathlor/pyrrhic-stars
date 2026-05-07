import type { LogEntry, ParticipantLog, InstanceLog, EncounterStats } from "../types";

export function makeEntry(overrides: Partial<LogEntry> = {}): LogEntry {
  return {
    group_id: "g1",
    instance_id: "i1",
    encounter_id: "boss",
    tick: 0,
    timestamp_ms: 0,
    source: "player_1",
    source_class: "gunner",
    target: "enemy_1",
    event_type: 1,
    ability_id: "slash",
    amount: 100,
    overkill: 0,
    school: "physical",
    is_crit: false,
    is_dodged: false,
    phase: "phase_1",
    boss_health: 1.0,
    pos_x: 0,
    pos_y: 0,
    pos_z: 0,
    resource_type: "",
    resource_delta: 0,
    resource_after: 0,
    ...overrides,
  };
}

export function makeParticipant(overrides: Partial<ParticipantLog> = {}): ParticipantLog {
  return {
    instance_id: "i1",
    entity_id: "player_1",
    name: "Alice",
    class: "gunner",
    is_bot: false,
    ...overrides,
  };
}

export function makeInstance(overrides: Partial<InstanceLog> = {}): InstanceLog {
  return {
    instance_id: "i1",
    group_id: "g1",
    encounter_id: "arena_boss",
    started_at: "2026-01-01T00:00:00Z",
    duration_ms: 60000,
    outcome: "player_win",
    source: "simulation",
    participants: [
      makeParticipant({ entity_id: "player_1", name: "Alice", class: "gunner" }),
      makeParticipant({ entity_id: "player_2", name: "Bob", class: "vanguard" }),
    ],
    ...overrides,
  };
}

export function makeEncounterStats(overrides: Partial<EncounterStats> = {}): EncounterStats {
  return {
    instance_damage: {},
    instance_healing: {},
    instance_deaths: {},
    instance_phases: {},
    boss_abilities: [],
    ...overrides,
  };
}

/** Standard 2-player participant list */
export const defaultParticipants: ParticipantLog[] = [
  makeParticipant({ entity_id: "player_1", name: "Alice", class: "gunner" }),
  makeParticipant({ entity_id: "player_2", name: "Bob", class: "vanguard" }),
  makeParticipant({ entity_id: "enemy_1", name: "Boss", class: "" }),
];
