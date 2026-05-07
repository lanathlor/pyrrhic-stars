import { computeDamageDone, computeDamageTaken } from "./damage";
import { makeEntry, defaultParticipants } from "../test/fixtures";
import { EVENT_TYPES } from "../constants";

describe("computeDamageDone", () => {
  it("returns empty array for no events", () => {
    expect(computeDamageDone([], defaultParticipants, 10000)).toEqual([]);
  });

  it("aggregates damage per player", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 200, ability_id: "slash" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 300, ability_id: "slash" }),
      makeEntry({ source: "player_2", event_type: EVENT_TYPES.DAMAGE, amount: 150, ability_id: "shield_bash" }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result).toHaveLength(2);
    expect(result[0].entityId).toBe("player_1");
    expect(result[0].totalDamage).toBe(500);
    expect(result[0].dps).toBeCloseTo(50);
    expect(result[1].entityId).toBe("player_2");
    expect(result[1].totalDamage).toBe(150);
  });

  it("excludes enemy damage from player totals", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100 }),
      makeEntry({ source: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 999, target: "player_1" }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result).toHaveLength(1);
    expect(result[0].totalDamage).toBe(100);
  });

  it("tracks crits per ability", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, is_crit: true, ability_id: "slash" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 50, is_crit: false, ability_id: "slash" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 200, is_crit: true, ability_id: "fireball" }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result[0].critRate).toBeCloseTo(2 / 3);
    expect(result[0].hitCount).toBe(3);
    expect(result[0].abilities).toHaveLength(2);

    const fireball = result[0].abilities.find((a) => a.abilityId === "fireball")!;
    expect(fireball.critRate).toBe(1);
    expect(fireball.maxHit).toBe(200);
  });

  it("sorts abilities by total damage descending", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 50, ability_id: "weak" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 300, ability_id: "strong" }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result[0].abilities[0].abilityId).toBe("strong");
  });

  it("uses auto_attack for empty ability_id", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, ability_id: "" }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result[0].abilities[0].abilityId).toBe("auto_attack");
  });

  it("falls back to entityId when participant not found", () => {
    const events = [
      makeEntry({ source: "player_99", event_type: EVENT_TYPES.DAMAGE, amount: 100 }),
    ];
    const result = computeDamageDone(events, defaultParticipants, 10000);
    expect(result[0].name).toBe("player_99");
    expect(result[0].className).toBe("");
  });

  it("ignores non-damage events", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 999 }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DEATH, amount: 0 }),
    ];
    expect(computeDamageDone(events, defaultParticipants, 10000)).toEqual([]);
  });
});

describe("computeDamageTaken", () => {
  it("returns empty array for no events", () => {
    expect(computeDamageTaken([], defaultParticipants, 10000)).toEqual([]);
  });

  it("aggregates damage taken per player target", () => {
    const events = [
      makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 200, ability_id: "cleave" }),
      makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, ability_id: "cleave" }),
      makeEntry({ source: "enemy_1", target: "player_2", event_type: EVENT_TYPES.DAMAGE, amount: 50, ability_id: "slam" }),
    ];
    const result = computeDamageTaken(events, defaultParticipants, 10000);
    expect(result).toHaveLength(2);
    expect(result[0].entityId).toBe("player_1");
    expect(result[0].totalDamageTaken).toBe(300);
    expect(result[0].dtps).toBeCloseTo(30);
    expect(result[1].totalDamageTaken).toBe(50);
  });

  it("breaks down by source and ability", () => {
    const events = [
      makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, ability_id: "cleave" }),
      makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 200, ability_id: "slam" }),
    ];
    const result = computeDamageTaken(events, defaultParticipants, 10000);
    expect(result[0].sources).toHaveLength(2);
    expect(result[0].sources[0].totalDamage).toBe(200); // slam sorted first (higher)
    expect(result[0].sources[0].abilityId).toBe("slam");
  });

  it("ignores damage not targeting a player", () => {
    const events = [
      makeEntry({ source: "player_1", target: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 500 }),
    ];
    expect(computeDamageTaken(events, defaultParticipants, 10000)).toEqual([]);
  });

  it("uses auto_attack for empty ability_id", () => {
    const events = [
      makeEntry({ source: "enemy_1", target: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, ability_id: "" }),
    ];
    const result = computeDamageTaken(events, defaultParticipants, 10000);
    expect(result[0].sources[0].abilityId).toBe("auto_attack");
  });
});
