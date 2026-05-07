import { screen } from "@testing-library/react";
import { DamageTakenTab } from "./DamageTakenTab";
import { renderWithFight } from "../../../test/renderWithFight";

describe("DamageTakenTab", () => {
  it("renders damage taken table headers", () => {
    renderWithFight(<DamageTakenTab />);
    expect(screen.getByText("Player")).toBeInTheDocument();
    expect(screen.getByText("Total Taken")).toBeInTheDocument();
    expect(screen.getByText("DTPS")).toBeInTheDocument();
  });

  it("renders players who took damage", () => {
    renderWithFight(<DamageTakenTab />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });
});
