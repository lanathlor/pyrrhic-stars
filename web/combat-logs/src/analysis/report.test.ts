import {
  groupByEncounter,
  computeReportOverview,
  computeProfileStats,
  computeCompStats,
  computeDurationHistogram,
  computeClassDPSStats,
  computeClassHPSStats,
  computeDeathStats,
  computeRaidDPSStats,
  computePhaseReach,
  computeWipePhases,
  computeProfileCombatStats,
} from "./report";
import { makeInstance, makeParticipant, makeEncounterStats } from "../test/fixtures";
import type { InstanceLog } from "../types";

// ── Helper: build N instances with varying outcomes ──

function makeRuns(count: number, overrides: Partial<InstanceLog> = {}): InstanceLog[] {
  return Array.from({ length: count }, (_, i) =>
    makeInstance({
      instance_id: `inst_${i}`,
      started_at: `2026-01-0${(i % 9) + 1}T00:00:00Z`,
      duration_ms: 30000 + i * 1000,
      ...overrides,
    })
  );
}

describe("groupByEncounter", () => {
  it("returns empty for no instances", () => {
    expect(groupByEncounter([])).toEqual([]);
  });

  it("groups by encounter_id", () => {
    const instances = [
      makeInstance({ encounter_id: "boss_a" }),
      makeInstance({ encounter_id: "boss_a", instance_id: "i2" }),
      makeInstance({ encounter_id: "boss_b", instance_id: "i3" }),
    ];
    const result = groupByEncounter(instances);
    expect(result).toHaveLength(2);
    expect(result[0].totalRuns).toBe(2); // boss_a sorted first (more runs)
    expect(result[0].encounterId).toBe("boss_a");
  });

  it("counts outcomes correctly", () => {
    const instances = [
      makeInstance({ outcome: "player_win" }),
      makeInstance({ outcome: "boss_win", instance_id: "i2" }),
      makeInstance({ outcome: "timeout", instance_id: "i3" }),
    ];
    const result = groupByEncounter(instances);
    expect(result[0].wins).toBe(1);
    expect(result[0].losses).toBe(1);
    expect(result[0].timeouts).toBe(1);
    expect(result[0].winRate).toBeCloseTo(1 / 3);
  });

  it("collects distinct profiles and classes", () => {
    const instances = [
      makeInstance({
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
          makeParticipant({ entity_id: "player_2", class: "vanguard", bot_profile: "sweaty" }),
        ],
      }),
      makeInstance({
        instance_id: "i2",
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "bad" }),
          makeParticipant({ entity_id: "player_2", class: "blade_dancer", bot_profile: "bad" }),
        ],
      }),
    ];
    const result = groupByEncounter(instances);
    expect(result[0].profiles).toEqual(["bad", "sweaty"]);
    expect(result[0].classes).toEqual(["blade_dancer", "gunner", "vanguard"]);
  });
});

describe("computeReportOverview", () => {
  it("computes overview stats", () => {
    const instances = [
      makeInstance({ outcome: "player_win", duration_ms: 30000 }),
      makeInstance({ instance_id: "i2", outcome: "boss_win", duration_ms: 50000 }),
      makeInstance({ instance_id: "i3", outcome: "player_win", duration_ms: 40000 }),
    ];
    const result = computeReportOverview(instances);
    expect(result.totalRuns).toBe(3);
    expect(result.wins).toBe(2);
    expect(result.losses).toBe(1);
    expect(result.winRate).toBeCloseTo(2 / 3);
    expect(result.durationStats.min).toBe(30000);
    expect(result.durationStats.max).toBe(50000);
    expect(result.durationStats.avg).toBeCloseTo(40000);
  });

  it("handles empty instances gracefully", () => {
    const result = computeReportOverview([makeInstance()]);
    expect(result.totalRuns).toBe(1);
  });
});

describe("computeProfileStats", () => {
  it("groups by dominant bot profile", () => {
    const instances = [
      makeInstance({
        participants: [
          makeParticipant({ entity_id: "player_1", bot_profile: "sweaty" }),
          makeParticipant({ entity_id: "player_2", bot_profile: "sweaty" }),
        ],
      }),
      makeInstance({
        instance_id: "i2",
        participants: [
          makeParticipant({ entity_id: "player_1", bot_profile: "bad" }),
          makeParticipant({ entity_id: "player_2", bot_profile: "bad" }),
        ],
      }),
    ];
    const result = computeProfileStats(instances);
    expect(result).toHaveLength(2);
    // sweaty first, bad second (ordering)
    expect(result[0].profile).toBe("sweaty");
    expect(result[1].profile).toBe("bad");
  });

  it("orders sweaty > average > bad", () => {
    const mkInst = (profile: string, id: string) =>
      makeInstance({
        instance_id: id,
        participants: [makeParticipant({ entity_id: "player_1", bot_profile: profile })],
      });
    const instances = [mkInst("bad", "i1"), mkInst("sweaty", "i2"), mkInst("average", "i3")];
    const result = computeProfileStats(instances);
    expect(result.map((r) => r.profile)).toEqual(["sweaty", "average", "bad"]);
  });
});

describe("computeCompStats", () => {
  it("groups by class+profile composition", () => {
    const instances = [
      makeInstance({
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
          makeParticipant({ entity_id: "player_2", class: "vanguard", bot_profile: "sweaty" }),
        ],
      }),
      makeInstance({
        instance_id: "i2",
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
          makeParticipant({ entity_id: "player_2", class: "vanguard", bot_profile: "sweaty" }),
        ],
      }),
    ];
    const result = computeCompStats(instances);
    expect(result).toHaveLength(1);
    expect(result[0].runs).toBe(2);
  });

  it("sorts by win rate descending", () => {
    const instances = [
      makeInstance({
        outcome: "boss_win",
        participants: [makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "bad" })],
      }),
      makeInstance({
        instance_id: "i2",
        outcome: "player_win",
        participants: [makeParticipant({ entity_id: "player_1", class: "vanguard", bot_profile: "sweaty" })],
      }),
    ];
    const result = computeCompStats(instances);
    expect(result[0].winRate).toBe(1); // vanguard sweaty wins
    expect(result[1].winRate).toBe(0);
  });
});

describe("computeDurationHistogram", () => {
  it("returns empty for no values", () => {
    expect(computeDurationHistogram([])).toEqual([]);
  });

  it("returns single bucket for identical values", () => {
    const result = computeDurationHistogram([5000, 5000, 5000]);
    expect(result).toHaveLength(1);
    expect(result[0].count).toBe(3);
  });

  it("distributes values across buckets", () => {
    const values = Array.from({ length: 100 }, (_, i) => i * 1000);
    const result = computeDurationHistogram(values, 10);
    expect(result).toHaveLength(10);
    const totalCount = result.reduce((sum, b) => sum + b.count, 0);
    expect(totalCount).toBe(100);
  });

  it("respects custom bucket count", () => {
    const result = computeDurationHistogram([1000, 2000, 3000, 4000], 2);
    expect(result).toHaveLength(2);
  });
});

describe("computeClassDPSStats", () => {
  it("computes per-class DPS distributions", () => {
    const instances = [
      makeInstance({ instance_id: "i1", duration_ms: 10000 }),
      makeInstance({ instance_id: "i2", duration_ms: 10000 }),
    ];
    const stats = makeEncounterStats({
      instance_damage: {
        i1: { gunner: 5000, vanguard: 3000 },
        i2: { gunner: 6000, vanguard: 4000 },
      },
    });
    const result = computeClassDPSStats(stats, instances);
    expect(result).toHaveLength(2);
    // gunner has higher median DPS
    expect(result[0].className).toBe("gunner");
    expect(result[0].stats.avg).toBeCloseTo(550); // (500+600)/2
  });

  it("skips instances with zero duration", () => {
    const instances = [makeInstance({ instance_id: "i1", duration_ms: 0 })];
    const stats = makeEncounterStats({ instance_damage: { i1: { gunner: 5000 } } });
    expect(computeClassDPSStats(stats, instances)).toEqual([]);
  });
});

describe("computeClassHPSStats", () => {
  it("computes per-class HPS distributions", () => {
    const instances = [makeInstance({ instance_id: "i1", duration_ms: 10000 })];
    const stats = makeEncounterStats({
      instance_healing: { i1: { vanguard: 2000 } },
    });
    const result = computeClassHPSStats(stats, instances);
    expect(result).toHaveLength(1);
    expect(result[0].className).toBe("vanguard");
    expect(result[0].stats.avg).toBeCloseTo(200);
  });
});

describe("computeDeathStats", () => {
  it("computes death percentiles", () => {
    const instances = makeRuns(3);
    const stats = makeEncounterStats({
      instance_deaths: { inst_0: 0, inst_1: 2, inst_2: 5 },
    });
    const result = computeDeathStats(stats, instances);
    expect(result.min).toBe(0);
    expect(result.max).toBe(5);
    expect(result.avg).toBeCloseTo(7 / 3);
  });

  it("defaults missing instance deaths to 0", () => {
    const instances = [makeInstance({ instance_id: "i1" })];
    const stats = makeEncounterStats(); // no deaths data
    const result = computeDeathStats(stats, instances);
    expect(result.min).toBe(0);
    expect(result.max).toBe(0);
  });
});

describe("computeRaidDPSStats", () => {
  it("computes raid-wide DPS per run", () => {
    const instances = [
      makeInstance({ instance_id: "i1", duration_ms: 10000 }),
      makeInstance({ instance_id: "i2", duration_ms: 20000 }),
    ];
    const stats = makeEncounterStats({
      instance_damage: {
        i1: { gunner: 3000, vanguard: 2000 }, // total 5000 / 10s = 500 DPS
        i2: { gunner: 8000, vanguard: 4000 }, // total 12000 / 20s = 600 DPS
      },
    });
    const result = computeRaidDPSStats(stats, instances);
    expect(result.min).toBeCloseTo(500);
    expect(result.max).toBeCloseTo(600);
    expect(result.avg).toBeCloseTo(550);
  });
});

describe("computePhaseReach", () => {
  it("returns empty for no instances", () => {
    expect(computePhaseReach(makeEncounterStats(), [])).toEqual([]);
  });

  it("computes cumulative phase reach", () => {
    const instances = makeRuns(3);
    const stats = makeEncounterStats({
      instance_phases: {
        inst_0: "phase_1",
        inst_1: "phase_2",
        inst_2: "phase_3",
      },
    });
    const result = computePhaseReach(stats, instances);
    expect(result).toHaveLength(3);
    // All 3 reached phase_1
    expect(result[0].phase).toBe("phase_1");
    expect(result[0].rate).toBeCloseTo(1);
    // 2 reached phase_2
    expect(result[1].rate).toBeCloseTo(2 / 3);
    // 1 reached phase_3
    expect(result[2].rate).toBeCloseTo(1 / 3);
  });
});

describe("computeWipePhases", () => {
  it("only considers boss_win outcomes", () => {
    const instances = [
      makeInstance({ outcome: "player_win", instance_id: "i1" }),
      makeInstance({ outcome: "boss_win", instance_id: "i2" }),
      makeInstance({ outcome: "boss_win", instance_id: "i3" }),
    ];
    const stats = makeEncounterStats({
      instance_phases: { i1: "phase_3", i2: "phase_1", i3: "phase_2" },
    });
    const result = computeWipePhases(stats, instances);
    expect(result).toHaveLength(2);
    expect(result.find((w) => w.phase === "phase_1")!.count).toBe(1);
    expect(result.find((w) => w.phase === "phase_2")!.count).toBe(1);
  });

  it("returns empty when no losses", () => {
    const instances = [makeInstance({ outcome: "player_win" })];
    const stats = makeEncounterStats({ instance_phases: { i1: "phase_3" } });
    expect(computeWipePhases(stats, instances)).toEqual([]);
  });
});

describe("computeProfileCombatStats", () => {
  it("produces combined profile + combat stats", () => {
    const instances = [
      makeInstance({
        instance_id: "i1",
        outcome: "player_win",
        duration_ms: 30000,
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
        ],
      }),
      makeInstance({
        instance_id: "i2",
        outcome: "boss_win",
        duration_ms: 50000,
        participants: [
          makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
        ],
      }),
    ];
    const stats = makeEncounterStats({
      instance_damage: { i1: { gunner: 5000 }, i2: { gunner: 8000 } },
      instance_deaths: { i1: 0, i2: 3 },
      instance_phases: { i1: "phase_3", i2: "phase_2" },
    });
    const result = computeProfileCombatStats(stats, instances);
    expect(result).toHaveLength(1);
    expect(result[0].profile).toBe("sweaty");
    expect(result[0].runs).toBe(2);
    expect(result[0].wins).toBe(1);
    expect(result[0].classDPS).toHaveLength(1);
    expect(result[0].classDPS[0].className).toBe("gunner");
    expect(result[0].deathStats.avg).toBeCloseTo(1.5);
  });
});
