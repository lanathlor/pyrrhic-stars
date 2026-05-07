import { fetchInstances, fetchInstance, fetchEvents, fetchEncounterStats, exportURL } from "./api";

const mockFetch = vi.fn();
beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch);
  mockFetch.mockReset();
});
afterEach(() => {
  vi.unstubAllGlobals();
});

function mockOk(data: unknown) {
  mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(data) });
}

function mockError(status: number) {
  mockFetch.mockResolvedValueOnce({ ok: false, status });
}

describe("fetchInstances", () => {
  it("fetches instances without params", async () => {
    mockOk([{ instance_id: "i1" }]);
    const result = await fetchInstances();
    expect(result).toEqual([{ instance_id: "i1" }]);
    expect(mockFetch).toHaveBeenCalledWith("/api/v1/logs/instances");
  });

  it("appends query params", async () => {
    mockOk([]);
    await fetchInstances({ source: "simulation", limit: "100" });
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("source=simulation");
    expect(url).toContain("limit=100");
  });

  it("throws on error status", async () => {
    mockError(500);
    await expect(fetchInstances()).rejects.toThrow("Failed to fetch instances: 500");
  });

  it("omits empty param values", async () => {
    mockOk([]);
    await fetchInstances({ source: "", limit: "10" });
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).not.toContain("source");
    expect(url).toContain("limit=10");
  });
});

describe("fetchInstance", () => {
  it("fetches a single instance", async () => {
    mockOk({ instance_id: "abc" });
    const result = await fetchInstance("abc");
    expect(result).toEqual({ instance_id: "abc" });
    expect(mockFetch).toHaveBeenCalledWith("/api/v1/logs/instances/abc");
  });

  it("throws on error", async () => {
    mockError(404);
    await expect(fetchInstance("nope")).rejects.toThrow("Failed to fetch instance: 404");
  });
});

describe("fetchEvents", () => {
  it("fetches events for an instance", async () => {
    mockOk([{ event_type: 1 }]);
    const result = await fetchEvents("abc");
    expect(result).toEqual([{ event_type: 1 }]);
    expect(mockFetch).toHaveBeenCalledWith("/api/v1/logs/instances/abc/events");
  });

  it("appends query params", async () => {
    mockOk([]);
    await fetchEvents("abc", { phase: "phase_1" });
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("phase=phase_1");
  });

  it("throws on error", async () => {
    mockError(500);
    await expect(fetchEvents("abc")).rejects.toThrow("Failed to fetch events: 500");
  });
});

describe("fetchEncounterStats", () => {
  it("fetches encounter stats", async () => {
    mockOk({ instance_damage: {} });
    const result = await fetchEncounterStats("boss_1");
    expect(result).toEqual({ instance_damage: {} });
    expect(mockFetch).toHaveBeenCalledWith("/api/v1/logs/stats/encounter/boss_1");
  });

  it("appends query params", async () => {
    mockOk({});
    await fetchEncounterStats("boss_1", { source: "simulation" });
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain("source=simulation");
  });

  it("throws on error", async () => {
    mockError(503);
    await expect(fetchEncounterStats("x")).rejects.toThrow("Failed to fetch encounter stats: 503");
  });
});

describe("exportURL", () => {
  it("returns the export URL for an instance", () => {
    expect(exportURL("abc-123")).toBe("/api/v1/logs/instances/abc-123/export");
  });
});
