import { computeDeathReports } from "./deaths";
import { makeEntry, defaultParticipants } from "../test/fixtures";
import { EVENT_TYPES } from "../constants";

describe("computeDeathReports", () => {
  it("returns empty for no events", () => {
    expect(computeDeathReports([], defaultParticipants)).toEqual([]);
  });

  it("reports a basic player death", () => {
    const events = [
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "player_1", source: "enemy_1", timestamp_ms: 5000, tick: 100 }),
    ];
    const result = computeDeathReports(events, defaultParticipants);
    expect(result).toHaveLength(1);
    expect(result[0].victim).toBe("player_1");
    expect(result[0].victimName).toBe("Alice");
    expect(result[0].victimClass).toBe("gunner");
    expect(result[0].timestampMs).toBe(5000);
  });

  it("finds the killing blow", () => {
    const events = [
      makeEntry({ event_type: EVENT_TYPES.DAMAGE, source: "enemy_1", target: "player_1", amount: 100, timestamp_ms: 4000, ability_id: "cleave" }),
      makeEntry({ event_type: EVENT_TYPES.DAMAGE, source: "enemy_1", target: "player_1", amount: 500, timestamp_ms: 4900, ability_id: "slam" }),
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "player_1", source: "enemy_1", timestamp_ms: 5000, tick: 100 }),
    ];
    const result = computeDeathReports(events, defaultParticipants);
    expect(result[0].killingBlow).not.toBeNull();
    expect(result[0].killingBlow!.ability_id).toBe("slam");
    expect(result[0].killingBlow!.amount).toBe(500);
  });

  it("collects up to 10 leadup events", () => {
    const leadupEvents = Array.from({ length: 15 }, (_, i) =>
      makeEntry({
        event_type: EVENT_TYPES.DAMAGE,
        source: "enemy_1",
        target: "player_1",
        amount: 10 + i,
        timestamp_ms: i * 100,
        tick: i,
      })
    );
    const deathEvent = makeEntry({
      event_type: EVENT_TYPES.DEATH,
      target: "player_1",
      source: "enemy_1",
      timestamp_ms: 2000,
      tick: 20,
    });
    const result = computeDeathReports([...leadupEvents, deathEvent], defaultParticipants);
    expect(result[0].leadup).toHaveLength(10);
  });

  it("ignores enemy deaths", () => {
    const events = [
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "enemy_1", source: "enemy_1", timestamp_ms: 5000 }),
    ];
    expect(computeDeathReports(events, defaultParticipants)).toEqual([]);
  });

  it("uses source as victim when target is empty", () => {
    const events = [
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "", source: "player_2", timestamp_ms: 5000 }),
    ];
    const result = computeDeathReports(events, defaultParticipants);
    expect(result).toHaveLength(1);
    expect(result[0].victim).toBe("player_2");
    expect(result[0].victimName).toBe("Bob");
  });

  it("sorts deaths chronologically", () => {
    const events = [
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "player_2", source: "enemy_1", timestamp_ms: 8000, tick: 2 }),
      makeEntry({ event_type: EVENT_TYPES.DEATH, target: "player_1", source: "enemy_1", timestamp_ms: 3000, tick: 1 }),
    ];
    const result = computeDeathReports(events, defaultParticipants);
    expect(result[0].victim).toBe("player_1");
    expect(result[1].victim).toBe("player_2");
  });
});
