import { screen } from "@testing-library/react";
import { TimelineTab } from "./TimelineTab";
import { renderWithFight } from "../../../test/renderWithFight";

// Mock Recharts ResponsiveContainer
vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart-container">{children}</div>
    ),
  };
});

describe("TimelineTab", () => {
  it("renders Boss Health section", () => {
    renderWithFight(<TimelineTab />);
    expect(screen.getByText("Boss Health")).toBeInTheDocument();
  });

  it("renders DPS Over Time section", () => {
    renderWithFight(<TimelineTab />);
    expect(screen.getByText("DPS Over Time")).toBeInTheDocument();
  });

  it("renders chart containers", () => {
    renderWithFight(<TimelineTab />);
    const containers = screen.getAllByTestId("chart-container");
    expect(containers.length).toBeGreaterThanOrEqual(2);
  });
});
