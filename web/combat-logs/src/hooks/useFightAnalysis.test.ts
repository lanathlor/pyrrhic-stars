import { normalizeEvents } from "./useFightAnalysis";
import { makeEntry } from "../test/fixtures";

describe("normalizeEvents", () => {
  it("returns empty for no events", () => {
    const { normalized, durationMs } = normalizeEvents([]);
    expect(normalized).toEqual([]);
    expect(durationMs).toBe(0);
  });

  it("returns events as-is when already fight-relative (starts at 0)", () => {
    const events = [
      makeEntry({ timestamp_ms: 0 }),
      makeEntry({ timestamp_ms: 5000 }),
      makeEntry({ timestamp_ms: 10000 }),
    ];
    const { normalized, durationMs } = normalizeEvents(events);
    expect(normalized).toBe(events); // same reference
    expect(durationMs).toBe(10000);
  });

  it("shifts timestamps when first event is non-zero", () => {
    const events = [
      makeEntry({ timestamp_ms: 1000 }),
      makeEntry({ timestamp_ms: 3000 }),
      makeEntry({ timestamp_ms: 6000 }),
    ];
    const { normalized, durationMs } = normalizeEvents(events);
    expect(normalized[0].timestamp_ms).toBe(0);
    expect(normalized[1].timestamp_ms).toBe(2000);
    expect(normalized[2].timestamp_ms).toBe(5000);
    expect(durationMs).toBe(5000);
  });

  it("returns durationMs of at least 1 for single-event streams", () => {
    const events = [makeEntry({ timestamp_ms: 0 })];
    const { durationMs } = normalizeEvents(events);
    expect(durationMs).toBe(1);
  });

  it("does not mutate original events", () => {
    const events = [
      makeEntry({ timestamp_ms: 500 }),
      makeEntry({ timestamp_ms: 1500 }),
    ];
    normalizeEvents(events);
    expect(events[0].timestamp_ms).toBe(500);
  });
});
