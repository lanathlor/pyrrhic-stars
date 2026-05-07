import { computeSummaryKPIs } from "./summary";
import { makeEntry, defaultParticipants as participants } from "../test/fixtures";

describe("computeSummaryKPIs", () => {
  it("returns zeroed stats for empty events", () => {
    const result = computeSummaryKPIs([], participants, 10000);
    expect(result.totalDamage).toBe(0);
    expect(result.totalHealing).toBe(0);
    expect(result.deathCount).toBe(0);
    expect(result.playerCount).toBe(2);
    expect(result.phases).toHaveLength(0);
  });

  it("sums player damage correctly", () => {
    const events = [
      makeEntry({ source: "player_1", event_type: 1, amount: 200 }),
      makeEntry({ source: "player_2", event_type: 1, amount: 300 }),
      makeEntry({ source: "enemy_1", event_type: 1, amount: 50 }), // enemy damage excluded from total
    ];
    const result = computeSummaryKPIs(events, participants, 10000);
    expect(result.totalDamage).toBe(500);
    expect(result.raidDps).toBeCloseTo(50); // 500 / 10s
  });

  it("counts player deaths", () => {
    const events = [
      makeEntry({ event_type: 11, target: "player_1", source: "enemy_1" }),
      makeEntry({ event_type: 11, target: "enemy_1", source: "player_1" }), // enemy death — has player source
    ];
    const result = computeSummaryKPIs(events, participants, 10000);
    expect(result.deathCount).toBe(2); // both have player in source or target
  });

  it("detects phases from events", () => {
    const events = [
      makeEntry({ phase: "phase_1" }),
      makeEntry({ phase: "phase_2" }),
      makeEntry({ phase: "phase_1" }),
    ];
    const result = computeSummaryKPIs(events, participants, 10000);
    expect(result.phases).toContain("phase_1");
    expect(result.phases).toContain("phase_2");
    expect(result.phases).toHaveLength(2);
  });

  it("sums healing", () => {
    const events = [
      makeEntry({ event_type: 2, amount: 150 }),
      makeEntry({ event_type: 2, amount: 250 }),
    ];
    const result = computeSummaryKPIs(events, participants, 10000);
    expect(result.totalHealing).toBe(400);
    expect(result.raidHps).toBeCloseTo(40);
  });
});
