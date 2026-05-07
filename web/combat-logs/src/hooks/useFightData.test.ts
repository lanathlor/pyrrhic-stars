import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useFightData } from "./useFightData";
import type { ReactNode } from "react";
import { createElement } from "react";

// Mock the api module
vi.mock("../api", () => ({
  fetchInstance: vi.fn(),
  fetchEvents: vi.fn(),
}));

import { fetchInstance, fetchEvents } from "../api";
const mockFetchInstance = vi.mocked(fetchInstance);
const mockFetchEvents = vi.mocked(fetchEvents);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useFightData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns loading state initially", () => {
    mockFetchInstance.mockReturnValue(new Promise(() => {})); // never resolves
    mockFetchEvents.mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useFightData("i1"), { wrapper: createWrapper() });
    expect(result.current.loading).toBe(true);
    expect(result.current.instance).toBeNull();
    expect(result.current.events).toEqual([]);
    expect(result.current.error).toBe("");
  });

  it("returns data after fetch resolves", async () => {
    const instance = { instance_id: "i1", group_id: "g1", encounter_id: "boss", started_at: "", duration_ms: 1000, outcome: "player_win", source: "sim", participants: [] };
    mockFetchInstance.mockResolvedValue(instance as never);
    mockFetchEvents.mockResolvedValue([{ event_type: 1 }] as never);

    const { result } = renderHook(() => useFightData("i1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.instance).toEqual(instance);
    expect(result.current.events).toEqual([{ event_type: 1 }]);
    expect(result.current.error).toBe("");
  });

  it("returns error message on fetch failure", async () => {
    mockFetchInstance.mockRejectedValue(new Error("Network error"));
    mockFetchEvents.mockResolvedValue([] as never);

    const { result } = renderHook(() => useFightData("i1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("Network error");
  });
});
