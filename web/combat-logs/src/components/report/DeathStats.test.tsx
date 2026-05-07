import { render, screen } from "@testing-library/react";
import { DeathStats } from "./DeathStats";
import type { PercentileStats, BossAbilityStat } from "../../types";

vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart">{children}</div>
    ),
  };
});

const deathStats: PercentileStats = {
  min: 0, max: 5, avg: 2.5, median: 2, p95: 4, p99: 5,
  values: [0, 1, 2, 2, 3, 4, 5],
};

const abilities: BossAbilityStat[] = [
  { ability_id: "ground_slam", total_damage: 80000, hits: 100, kills: 30, dodges: 20 },
  { ability_id: "cleave", total_damage: 50000, hits: 200, kills: 5, dodges: 40 },
];

describe("DeathStats", () => {
  it("renders heading", () => {
    render(<DeathStats deathStats={deathStats} bossAbilities={abilities} percentileMode="p95" />);
    expect(screen.getByText("Deaths per Run")).toBeInTheDocument();
  });

  it("renders median, avg, and percentile values", () => {
    render(<DeathStats deathStats={deathStats} bossAbilities={abilities} percentileMode="p95" />);
    expect(screen.getByText("2.0")).toBeInTheDocument(); // median
    expect(screen.getByText("2.5")).toBeInTheDocument(); // avg
    expect(screen.getByText("4")).toBeInTheDocument(); // p95
  });

  it("renders most lethal abilities", () => {
    render(<DeathStats deathStats={deathStats} bossAbilities={abilities} percentileMode="p95" />);
    expect(screen.getByText("Most Lethal Abilities")).toBeInTheDocument();
    expect(screen.getByText("Ground Slam")).toBeInTheDocument();
    expect(screen.getByText("Cleave")).toBeInTheDocument();
  });

  it("renders chart container", () => {
    render(<DeathStats deathStats={deathStats} bossAbilities={abilities} percentileMode="p95" />);
    expect(screen.getByTestId("chart")).toBeInTheDocument();
  });
});
