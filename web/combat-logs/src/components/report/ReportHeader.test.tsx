import { render, screen } from "@testing-library/react";
import { ReportHeader } from "./ReportHeader";
import type { ReportOverview, PercentileStats } from "../../types";

const stats: PercentileStats = {
  min: 20000, max: 80000, avg: 45000, median: 42000, p95: 70000, p99: 78000,
  values: [20000, 30000, 42000, 50000, 70000, 78000, 80000],
};

const overview: ReportOverview = {
  encounterId: "arena_boss",
  totalRuns: 100,
  wins: 60,
  losses: 30,
  timeouts: 10,
  winRate: 0.6,
  durationStats: stats,
  firstRun: "2026-01-01T00:00:00Z",
  lastRun: "2026-01-07T00:00:00Z",
};

describe("ReportHeader", () => {
  it("renders encounter name", () => {
    render(<ReportHeader overview={overview} percentileMode="p95" />);
    expect(screen.getByText("arena boss")).toBeInTheDocument();
  });

  it("renders run count", () => {
    render(<ReportHeader overview={overview} percentileMode="p95" />);
    expect(screen.getByText(/100 simulation runs/)).toBeInTheDocument();
  });

  it("renders KPI cards", () => {
    render(<ReportHeader overview={overview} percentileMode="p95" />);
    expect(screen.getByText("Win Rate")).toBeInTheDocument();
    expect(screen.getByText("Total Runs")).toBeInTheDocument();
    expect(screen.getByText("Avg Duration")).toBeInTheDocument();
  });

  it("renders raid DPS when provided", () => {
    const raidDPS: PercentileStats = { ...stats, median: 500 };
    render(<ReportHeader overview={overview} raidDPSStats={raidDPS} percentileMode="p95" />);
    expect(screen.getByText("Raid DPS (med)")).toBeInTheDocument();
  });

  it("renders death stats when provided", () => {
    const deathStats: PercentileStats = { ...stats, median: 2.5, avg: 3.0 };
    render(<ReportHeader overview={overview} deathStats={deathStats} percentileMode="p95" />);
    expect(screen.getByText("Deaths/Run")).toBeInTheDocument();
  });

  it("switches percentile label based on mode", () => {
    render(<ReportHeader overview={overview} percentileMode="p99" />);
    expect(screen.getByText("Duration P99")).toBeInTheDocument();
  });
});
