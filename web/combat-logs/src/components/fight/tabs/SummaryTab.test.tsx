import { screen } from "@testing-library/react";
import { SummaryTab } from "./SummaryTab";
import { renderWithFight } from "../../../test/renderWithFight";

describe("SummaryTab", () => {
  it("renders KPI cards", () => {
    renderWithFight(<SummaryTab />);
    expect(screen.getByText("Total Damage")).toBeInTheDocument();
    expect(screen.getByText("Raid DPS")).toBeInTheDocument();
    expect(screen.getByText("Total Healing")).toBeInTheDocument();
    expect(screen.getByText("Deaths")).toBeInTheDocument();
    expect(screen.getByText("Duration")).toBeInTheDocument();
    expect(screen.getByText("Players")).toBeInTheDocument();
  });

  it("renders DPS Rankings section", () => {
    renderWithFight(<SummaryTab />);
    expect(screen.getByText("DPS Rankings")).toBeInTheDocument();
    // Both players appear (possibly multiple times across sections)
    expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Bob").length).toBeGreaterThanOrEqual(1);
  });

  it("renders deaths section when there are deaths", () => {
    renderWithFight(<SummaryTab />);
    expect(screen.getByText(/Deaths \(1\)/)).toBeInTheDocument();
  });
});
