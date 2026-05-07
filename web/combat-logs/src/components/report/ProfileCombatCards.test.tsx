import { render, screen } from "@testing-library/react";
import { ProfileCombatCards } from "./ProfileCombatCards";
import type { ProfileCombatStats, PercentileStats, ClassDPSDistribution } from "../../types";

const dStats: PercentileStats = {
  min: 20000, max: 50000, avg: 35000, median: 34000, p95: 48000, p99: 49000,
  values: [20000, 34000, 50000],
};

const classDPS: ClassDPSDistribution[] = [
  { className: "gunner", stats: { min: 100, max: 800, avg: 500, median: 480, p95: 750, p99: 790, values: [100, 480, 800] } },
];

const profiles: ProfileCombatStats[] = [
  {
    profile: "sweaty",
    runs: 50,
    wins: 40,
    losses: 10,
    timeouts: 0,
    winRate: 0.8,
    avgDurationMs: 35000,
    durationStats: dStats,
    classDPS,
    deathStats: { min: 0, max: 5, avg: 2, median: 2, p95: 4, p99: 5, values: [0, 2, 5] },
    phaseReach: { phase_1: 1, phase_2: 0.8, phase_3: 0.4 },
  },
];

describe("ProfileCombatCards", () => {
  it("renders profile label and run count", () => {
    render(<ProfileCombatCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("Sweaty")).toBeInTheDocument();
    expect(screen.getByText("50 runs")).toBeInTheDocument();
  });

  it("renders win rate", () => {
    render(<ProfileCombatCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("80.0%")).toBeInTheDocument();
  });

  it("renders class DPS section", () => {
    render(<ProfileCombatCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("Median DPS by Class")).toBeInTheDocument();
    expect(screen.getByText("Gunner")).toBeInTheDocument();
  });

  it("renders death stats", () => {
    render(<ProfileCombatCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("Deaths/Run (med)")).toBeInTheDocument();
    expect(screen.getByText("Deaths/Run (avg)")).toBeInTheDocument();
  });

  it("renders phase reach when multiple phases", () => {
    render(<ProfileCombatCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("Phase Reach")).toBeInTheDocument();
    expect(screen.getByText(/P1: 100%/)).toBeInTheDocument();
    expect(screen.getByText(/P2: 80%/)).toBeInTheDocument();
  });
});
