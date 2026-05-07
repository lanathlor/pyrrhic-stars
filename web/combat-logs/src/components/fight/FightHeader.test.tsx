import { screen } from "@testing-library/react";
import { FightHeader } from "./FightHeader";
import { renderWithFight, fightInstance } from "../../test/renderWithFight";

describe("FightHeader", () => {
  it("renders encounter name", () => {
    renderWithFight(<FightHeader instance={fightInstance} />);
    expect(screen.getByText("arena_boss")).toBeInTheDocument();
  });

  it("renders outcome badge", () => {
    renderWithFight(<FightHeader instance={fightInstance} />);
    expect(screen.getByText("player win")).toBeInTheDocument();
  });

  it("renders participant list", () => {
    renderWithFight(<FightHeader instance={fightInstance} />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("renders class icons for players", () => {
    renderWithFight(<FightHeader instance={fightInstance} />);
    expect(screen.getByText("Gunner")).toBeInTheDocument();
    expect(screen.getByText("Vanguard")).toBeInTheDocument();
  });
});
