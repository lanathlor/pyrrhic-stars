import { render } from "@testing-library/react";
import { FightProvider } from "../hooks/FightContext";
import { makeInstance, makeEntry, makeParticipant } from "./fixtures";
import type { InstanceLog, LogEntry } from "../types";
import { EVENT_TYPES } from "../constants";

/** Standard fixture data for fight component tests */
export const fightInstance: InstanceLog = makeInstance({
  instance_id: "test-fight",
  encounter_id: "arena_boss",
  outcome: "player_win",
  duration_ms: 60000,
  started_at: "2026-01-01T12:00:00Z",
  participants: [
    makeParticipant({ entity_id: "player_1", name: "Alice", class: "gunner" }),
    makeParticipant({ entity_id: "player_2", name: "Bob", class: "vanguard" }),
    makeParticipant({ entity_id: "enemy_1", name: "Boss", class: "" }),
  ],
});

export const fightEvents: LogEntry[] = [
  // Damage events
  makeEntry({ source: "player_1", target: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 500, ability_id: "rapid_fire", timestamp_ms: 0, phase: "phase_1", boss_health: 1.0, school: "physical" }),
  makeEntry({ source: "player_2", target: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 300, ability_id: "shield_bash", timestamp_ms: 1000, phase: "phase_1", boss_health: 0.9, school: "physical" }),
  makeEntry({ source: "player_1", target: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 700, ability_id: "rapid_fire", is_crit: true, timestamp_ms: 3000, phase: "phase_1", boss_health: 0.8 }),
  // Boss damage to players
  makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 200, ability_id: "cleave", timestamp_ms: 2000, phase: "phase_1", boss_health: 0.85 }),
  makeEntry({ source: "enemy_1", target: "player_2", event_type: EVENT_TYPES.DAMAGE, amount: 150, ability_id: "slam", timestamp_ms: 4000, phase: "phase_1", boss_health: 0.7 }),
  // Healing
  makeEntry({ source: "player_2", target: "player_1", event_type: EVENT_TYPES.HEAL, amount: 100, ability_id: "bandage", timestamp_ms: 5000, phase: "phase_1", boss_health: 0.6 }),
  // Phase change
  makeEntry({ source: "", target: "", event_type: EVENT_TYPES.PHASE_CHANGE, amount: 0, ability_id: "", timestamp_ms: 6000, phase: "phase_2", boss_health: 0.5 }),
  // More damage in phase 2
  makeEntry({ source: "player_1", target: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 1000, ability_id: "ultimate", timestamp_ms: 8000, phase: "phase_2", boss_health: 0.3 }),
  // Death
  makeEntry({ source: "enemy_1", target: "player_2", event_type: EVENT_TYPES.DEATH, amount: 0, ability_id: "", timestamp_ms: 9000, phase: "phase_2", boss_health: 0.2 }),
];

export function renderWithFight(ui: React.ReactElement, opts?: { instance?: InstanceLog; events?: LogEntry[] }) {
  return render(
    <FightProvider instance={opts?.instance ?? fightInstance} events={opts?.events ?? fightEvents}>
      {ui}
    </FightProvider>
  );
}
