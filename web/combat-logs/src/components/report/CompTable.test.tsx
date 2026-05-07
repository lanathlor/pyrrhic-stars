import { render, screen } from "@testing-library/react";
import { CompTable } from "./CompTable";
import type { CompStats, PercentileStats } from "../../types";

const stats: PercentileStats = {
  min: 20000, max: 50000, avg: 35000, median: 34000, p95: 48000, p99: 49000,
  values: [20000, 34000, 48000, 49000, 50000],
};

const comps: CompStats[] = [
  {
    name: "gunner+vanguard (sweaty+sweaty)",
    classes: ["gunner", "vanguard"],
    profiles: ["sweaty", "sweaty"],
    runs: 50,
    wins: 35,
    losses: 15,
    timeouts: 0,
    winRate: 0.7,
    avgDurationMs: 35000,
    durationStats: stats,
  },
];

describe("CompTable", () => {
  it("renders table headers", () => {
    render(<CompTable comps={comps} percentileMode="p95" />);
    expect(screen.getByText("Composition")).toBeInTheDocument();
    expect(screen.getByText("Profiles")).toBeInTheDocument();
    expect(screen.getByText("Runs")).toBeInTheDocument();
    expect(screen.getByText("Win%")).toBeInTheDocument();
    expect(screen.getByText("Avg Duration")).toBeInTheDocument();
    expect(screen.getByText("Median")).toBeInTheDocument();
    expect(screen.getByText("P95")).toBeInTheDocument();
  });

  it("renders composition data", () => {
    render(<CompTable comps={comps} percentileMode="p95" />);
    expect(screen.getByText("50")).toBeInTheDocument(); // runs
    expect(screen.getByText("Gunner")).toBeInTheDocument();
    expect(screen.getByText("Vanguard")).toBeInTheDocument();
  });

  it("shows P99 header when in p99 mode", () => {
    render(<CompTable comps={comps} percentileMode="p99" />);
    expect(screen.getByText("P99")).toBeInTheDocument();
  });
});
