import { screen } from "@testing-library/react";
import { DamageDoneTab } from "./DamageDoneTab";
import { renderWithFight } from "../../../test/renderWithFight";

// Mock Recharts ResponsiveContainer (needs DOM measurements)
vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="responsive-container">{children}</div>
    ),
  };
});

describe("DamageDoneTab", () => {
  it("renders player damage table", () => {
    renderWithFight(<DamageDoneTab />);
    expect(screen.getByText("Player")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();
    expect(screen.getByText("DPS")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("renders pie chart container", () => {
    renderWithFight(<DamageDoneTab />);
    expect(screen.getByTestId("responsive-container")).toBeInTheDocument();
  });

  it("shows column headers", () => {
    renderWithFight(<DamageDoneTab />);
    expect(screen.getByText("Crit%")).toBeInTheDocument();
    expect(screen.getByText("Hits")).toBeInTheDocument();
    expect(screen.getByText("%")).toBeInTheDocument();
  });
});
