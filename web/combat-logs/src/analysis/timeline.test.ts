import { computeBossHPTimeline, computeDPSTimeline, computePhaseMarkers } from "./timeline";
import { makeEntry, makeParticipant } from "../test/fixtures";
import { EVENT_TYPES } from "../constants";

describe("computeBossHPTimeline", () => {
  it("returns empty for no events", () => {
    expect(computeBossHPTimeline([])).toEqual([]);
  });

  it("tracks HP changes as percentage", () => {
    const events = [
      makeEntry({ boss_health: 1.0, timestamp_ms: 0 }),
      makeEntry({ boss_health: 0.8, timestamp_ms: 1000 }),
      makeEntry({ boss_health: 0.5, timestamp_ms: 2000 }),
    ];
    const result = computeBossHPTimeline(events);
    expect(result).toHaveLength(3);
    expect(result[0].value).toBe(100);
    expect(result[1].value).toBe(80);
    expect(result[2].value).toBe(50);
  });

  it("skips duplicate HP values", () => {
    const events = [
      makeEntry({ boss_health: 1.0, timestamp_ms: 0 }),
      makeEntry({ boss_health: 1.0, timestamp_ms: 500 }),
      makeEntry({ boss_health: 0.9, timestamp_ms: 1000 }),
    ];
    const result = computeBossHPTimeline(events);
    expect(result).toHaveLength(2);
  });

  it("skips events with boss_health <= 0", () => {
    const events = [
      makeEntry({ boss_health: 0.5, timestamp_ms: 0 }),
      makeEntry({ boss_health: 0, timestamp_ms: 1000 }),
      makeEntry({ boss_health: -1, timestamp_ms: 2000 }),
    ];
    const result = computeBossHPTimeline(events);
    expect(result).toHaveLength(1);
  });

  it("downsamples when more than 300 points", () => {
    const events = Array.from({ length: 400 }, (_, i) =>
      makeEntry({ boss_health: 1 - i * 0.002, timestamp_ms: i * 100 })
    );
    const result = computeBossHPTimeline(events);
    expect(result.length).toBeLessThanOrEqual(302); // 300 + possible last point
  });
});

describe("computeDPSTimeline", () => {
  const participants = [
    makeParticipant({ entity_id: "player_1", name: "Alice" }),
    makeParticipant({ entity_id: "player_2", name: "Bob" }),
  ];

  it("returns empty for zero duration", () => {
    expect(computeDPSTimeline([], participants, 0)).toEqual([]);
  });

  it("returns empty for no player participants", () => {
    const enemyOnly = [makeParticipant({ entity_id: "enemy_1", name: "Boss" })];
    const events = [makeEntry({ event_type: EVENT_TYPES.DAMAGE, source: "enemy_1", amount: 100, timestamp_ms: 500 })];
    expect(computeDPSTimeline(events, enemyOnly, 10000)).toEqual([]);
  });

  it("produces DPS data points with player keys", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 1000, timestamp_ms: 500 }),
      makeEntry({ source: "player_2", event_type: EVENT_TYPES.DAMAGE, amount: 500, timestamp_ms: 500 }),
    ];
    const result = computeDPSTimeline(events, participants, 10000);
    expect(result.length).toBeGreaterThan(0);
    expect(result[0]).toHaveProperty("player_1");
    expect(result[0]).toHaveProperty("player_2");
    expect(result[0]).toHaveProperty("timestampMs");
  });

  it("ignores non-damage and non-player events", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.HEAL, amount: 999, timestamp_ms: 500 }),
      makeEntry({ source: "enemy_1", event_type: EVENT_TYPES.DAMAGE, amount: 999, timestamp_ms: 500 }),
    ];
    const result = computeDPSTimeline(events, participants, 10000);
    // All DPS values should be 0
    for (const point of result) {
      expect(point.player_1).toBe(0);
      expect(point.player_2).toBe(0);
    }
  });
});

describe("computePhaseMarkers", () => {
  it("returns empty for no events", () => {
    expect(computePhaseMarkers([], 10000)).toEqual([]);
  });

  it("tracks phase transitions", () => {
    const events = [
      makeEntry({ phase: "phase_1", timestamp_ms: 0 }),
      makeEntry({ phase: "phase_1", timestamp_ms: 2000 }),
      makeEntry({ phase: "phase_2", timestamp_ms: 5000 }),
      makeEntry({ phase: "phase_2", timestamp_ms: 8000 }),
    ];
    const result = computePhaseMarkers(events, 10000);
    expect(result).toHaveLength(2);
    expect(result[0]).toEqual({ phase: "phase_1", startMs: 0, endMs: 5000 });
    expect(result[1]).toEqual({ phase: "phase_2", startMs: 5000, endMs: 10000 });
  });

  it("handles single phase", () => {
    const events = [
      makeEntry({ phase: "phase_1", timestamp_ms: 0 }),
      makeEntry({ phase: "phase_1", timestamp_ms: 5000 }),
    ];
    const result = computePhaseMarkers(events, 10000);
    expect(result).toHaveLength(1);
    expect(result[0].endMs).toBe(10000);
  });

  it("handles three phases", () => {
    const events = [
      makeEntry({ phase: "phase_1", timestamp_ms: 0 }),
      makeEntry({ phase: "phase_2", timestamp_ms: 3000 }),
      makeEntry({ phase: "phase_3", timestamp_ms: 7000 }),
    ];
    const result = computePhaseMarkers(events, 10000);
    expect(result).toHaveLength(3);
    expect(result[0].endMs).toBe(3000);
    expect(result[1].endMs).toBe(7000);
    expect(result[2].endMs).toBe(10000);
  });

  it("skips events without phase", () => {
    const events = [
      makeEntry({ phase: "", timestamp_ms: 0 }),
      makeEntry({ phase: "phase_1", timestamp_ms: 2000 }),
    ];
    const result = computePhaseMarkers(events, 10000);
    expect(result).toHaveLength(1);
    expect(result[0].phase).toBe("phase_1");
  });
});
