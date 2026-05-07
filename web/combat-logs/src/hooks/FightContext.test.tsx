import { render, screen, renderHook } from "@testing-library/react";
import { FightProvider, useFightContext } from "./FightContext";
import { makeInstance, makeEntry } from "../test/fixtures";
import type { ReactNode } from "react";

const instance = makeInstance();
const events = [
  makeEntry({ timestamp_ms: 0, event_type: 1, source: "player_1", amount: 100 }),
  makeEntry({ timestamp_ms: 5000, event_type: 1, source: "player_2", amount: 200 }),
];

function Wrapper({ children }: { children: ReactNode }) {
  return (
    <FightProvider instance={instance} events={events}>
      {children}
    </FightProvider>
  );
}

describe("FightProvider", () => {
  it("renders children", () => {
    render(
      <FightProvider instance={instance} events={events}>
        <div data-testid="child">hello</div>
      </FightProvider>
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });
});

describe("useFightContext", () => {
  it("returns context with instance and analysis", () => {
    const { result } = renderHook(() => useFightContext(), { wrapper: Wrapper });
    expect(result.current.instance).toBe(instance);
    expect(result.current.events).toBe(events);
    expect(result.current.analysis).toBeDefined();
    expect(result.current.analysis.summary).toBeDefined();
    expect(result.current.selectedPhase).toBeNull();
    expect(result.current.selectedPlayer).toBeNull();
  });

  it("throws when used outside provider", () => {
    expect(() => {
      renderHook(() => useFightContext());
    }).toThrow("useFightContext must be used within FightProvider");
  });
});
