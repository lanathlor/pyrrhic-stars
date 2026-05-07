import { computeHealingDone } from "./healing";
import { makeEntry, defaultParticipants } from "../test/fixtures";
import { EVENT_TYPES } from "../constants";

describe("computeHealingDone", () => {
  it("returns empty array for no events", () => {
    expect(computeHealingDone([], defaultParticipants, 10000)).toEqual([]);
  });

  it("aggregates healing per source", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 100, ability_id: "heal" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 200, ability_id: "heal" }),
      makeEntry({ source: "player_2", event_type: EVENT_TYPES.HEAL, amount: 50, ability_id: "bandage" }),
    ];
    const result = computeHealingDone(events, defaultParticipants, 10000);
    expect(result).toHaveLength(2);
    expect(result[0].entityId).toBe("player_1");
    expect(result[0].totalHealing).toBe(300);
    expect(result[0].hps).toBeCloseTo(30);
    expect(result[1].totalHealing).toBe(50);
  });

  it("tracks crits per ability", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 100, is_crit: true, ability_id: "heal" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 80, is_crit: false, ability_id: "heal" }),
    ];
    const result = computeHealingDone(events, defaultParticipants, 10000);
    expect(result[0].critRate).toBeCloseTo(0.5);
    expect(result[0].abilities[0].critCount).toBe(1);
    expect(result[0].abilities[0].maxHit).toBe(100);
  });

  it("uses auto_heal for empty ability_id", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 100, ability_id: "" }),
    ];
    const result = computeHealingDone(events, defaultParticipants, 10000);
    expect(result[0].abilities[0].abilityId).toBe("auto_heal");
  });

  it("ignores non-heal events", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 500 }),
    ];
    expect(computeHealingDone(events, defaultParticipants, 10000)).toEqual([]);
  });

  it("includes enemy healing", () => {
    const events = [
      makeEntry({ source: "enemy_1", event_type: EVENT_TYPES.HEAL, amount: 999, ability_id: "regen" }),
    ];
    const result = computeHealingDone(events, defaultParticipants, 10000);
    expect(result).toHaveLength(1);
    expect(result[0].totalHealing).toBe(999);
  });

  it("sorts by total healing descending", () => {
    const events = [
      makeEntry({ source: "player_2", event_type: EVENT_TYPES.HEAL, amount: 500, ability_id: "big_heal" }),
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 100, ability_id: "small_heal" }),
    ];
    const result = computeHealingDone(events, defaultParticipants, 10000);
    expect(result[0].entityId).toBe("player_2");
  });
});
