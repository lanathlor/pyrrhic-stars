import { filterByPhase, filterByPlayer, filterByType } from "./filters";
import { makeEntry } from "../test/fixtures";

describe("filterByPhase", () => {
  const events = [
    makeEntry({ phase: "phase_1" }),
    makeEntry({ phase: "phase_2" }),
    makeEntry({ phase: "phase_1" }),
  ];

  it("returns all events when phase is null", () => {
    expect(filterByPhase(events, null)).toHaveLength(3);
  });

  it("filters to matching phase", () => {
    expect(filterByPhase(events, "phase_1")).toHaveLength(2);
    expect(filterByPhase(events, "phase_2")).toHaveLength(1);
  });

  it("returns empty for non-existent phase", () => {
    expect(filterByPhase(events, "phase_3")).toHaveLength(0);
  });
});

describe("filterByPlayer", () => {
  const events = [
    makeEntry({ source: "player_1", target: "enemy_1" }),
    makeEntry({ source: "player_2", target: "enemy_1" }),
    makeEntry({ source: "enemy_1", target: "player_1" }),
  ];

  it("includes events where player is source or target", () => {
    const result = filterByPlayer(events, "player_1");
    expect(result).toHaveLength(2);
  });

  it("returns empty for unknown entity", () => {
    expect(filterByPlayer(events, "player_99")).toHaveLength(0);
  });
});

describe("filterByType", () => {
  const events = [
    makeEntry({ event_type: 1 }),
    makeEntry({ event_type: 2 }),
    makeEntry({ event_type: 11 }),
    makeEntry({ event_type: 1 }),
  ];

  it("filters to specified types", () => {
    expect(filterByType(events, [1])).toHaveLength(2);
    expect(filterByType(events, [1, 11])).toHaveLength(3);
  });

  it("returns empty when no types match", () => {
    expect(filterByType(events, [99])).toHaveLength(0);
  });
});
