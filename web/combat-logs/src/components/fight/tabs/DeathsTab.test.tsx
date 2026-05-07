import { screen } from "@testing-library/react";
import { DeathsTab } from "./DeathsTab";
import { renderWithFight } from "../../../test/renderWithFight";
import { makeInstance, makeParticipant, makeEntry } from "../../../test/fixtures";
import { EVENT_TYPES } from "../../../constants";

describe("DeathsTab", () => {
  it("renders death reports", () => {
    renderWithFight(<DeathsTab />);
    // Bob died in our fixture events
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows empty state when no deaths", () => {
    const instance = makeInstance({
      participants: [makeParticipant({ entity_id: "player_1", name: "Alice", class: "gunner" })],
    });
    const events = [
      makeEntry({ source: "player_1", event_type: EVENT_TYPES.DAMAGE, amount: 100, timestamp_ms: 0 }),
    ];
    renderWithFight(<DeathsTab />, { instance, events });
    expect(screen.getByText("No player deaths recorded.")).toBeInTheDocument();
  });

  it("shows killing blow info", () => {
    renderWithFight(<DeathsTab />);
    expect(screen.getByText(/Killing blow/)).toBeInTheDocument();
  });
});
