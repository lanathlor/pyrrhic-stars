import { screen } from "@testing-library/react";
import { HealingTab } from "./HealingTab";
import { renderWithFight } from "../../../test/renderWithFight";
import { makeInstance, makeParticipant, makeEntry } from "../../../test/fixtures";
import { EVENT_TYPES } from "../../../constants";

describe("HealingTab", () => {
  it("renders healing table when healing exists", () => {
    renderWithFight(<HealingTab />);
    expect(screen.getByText("Player")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();
    expect(screen.getByText("HPS")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows empty state when no healing events", () => {
    const instance = makeInstance({
      participants: [makeParticipant({ entity_id: "player_1", name: "Alice", class: "gunner" })],
    });
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, timestamp_ms: 0 }),
    ];
    renderWithFight(<HealingTab />, { instance, events });
    expect(screen.getByText("No healing events recorded.")).toBeInTheDocument();
  });
});
