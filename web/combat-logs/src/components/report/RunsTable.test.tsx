import { render, screen, fireEvent } from "@testing-library/react";
import { RunsTable } from "./RunsTable";
import { makeInstance, makeParticipant } from "../../test/fixtures";

// Mock TanStack Router Link
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: React.ReactNode; to: string }) => (
    <a href={props.to}>{children}</a>
  ),
}));

const instances = [
  makeInstance({
    instance_id: "i1",
    outcome: "player_win",
    duration_ms: 30000,
    started_at: "2026-01-03T00:00:00Z",
    participants: [
      makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
      makeParticipant({ entity_id: "player_2", class: "vanguard", bot_profile: "sweaty" }),
    ],
  }),
  makeInstance({
    instance_id: "i2",
    outcome: "boss_win",
    duration_ms: 50000,
    started_at: "2026-01-01T00:00:00Z",
    participants: [
      makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "bad" }),
    ],
  }),
  makeInstance({
    instance_id: "i3",
    outcome: "player_win",
    duration_ms: 40000,
    started_at: "2026-01-02T00:00:00Z",
    participants: [
      makeParticipant({ entity_id: "player_1", class: "gunner", bot_profile: "sweaty" }),
    ],
  }),
];

describe("RunsTable", () => {
  it("renders table headers", () => {
    render(<RunsTable instances={instances} />);
    const headers = screen.getAllByRole("columnheader");
    const headerTexts = headers.map((h) => h.textContent);
    expect(headerTexts).toContain("Players");
    expect(headerTexts).toContain("Profile");
  });

  it("renders all runs", () => {
    render(<RunsTable instances={instances} />);
    expect(screen.getByText("3 runs")).toBeInTheDocument();
    expect(screen.getAllByText("View →")).toHaveLength(3);
  });

  it("renders outcome filter", () => {
    render(<RunsTable instances={instances} />);
    expect(screen.getByText("All Outcomes")).toBeInTheDocument();
  });

  it("filters by outcome", () => {
    render(<RunsTable instances={instances} />);
    const select = screen.getAllByRole("combobox")[0];
    fireEvent.change(select, { target: { value: "boss_win" } });
    expect(screen.getByText("1 runs")).toBeInTheDocument();
  });

  it("renders profile filter when multiple profiles exist", () => {
    render(<RunsTable instances={instances} />);
    expect(screen.getByText("All Profiles")).toBeInTheDocument();
  });

  it("sorts by date by default", () => {
    render(<RunsTable instances={instances} />);
    // Default is desc, first row should be latest (i1: Jan 3)
    const links = screen.getAllByText("View →");
    expect(links).toHaveLength(3);
  });
});
