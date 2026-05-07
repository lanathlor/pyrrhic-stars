import { render, screen } from "@testing-library/react";
import { ProfileCards } from "./ProfileCards";
import type { ProfileStats, PercentileStats } from "../../types";

const stats: PercentileStats = {
  min: 20000, max: 50000, avg: 35000, median: 34000, p95: 48000, p99: 49000,
  values: [20000, 34000, 50000],
};

const profiles: ProfileStats[] = [
  { profile: "sweaty", runs: 50, wins: 40, losses: 10, timeouts: 0, winRate: 0.8, avgDurationMs: 35000, durationStats: stats },
  { profile: "bad", runs: 30, wins: 5, losses: 25, timeouts: 0, winRate: 0.167, avgDurationMs: 55000, durationStats: stats },
];

describe("ProfileCards", () => {
  it("renders profile labels", () => {
    render(<ProfileCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("Sweaty")).toBeInTheDocument();
    expect(screen.getByText("Bad")).toBeInTheDocument();
  });

  it("renders run counts", () => {
    render(<ProfileCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("50 runs")).toBeInTheDocument();
    expect(screen.getByText("30 runs")).toBeInTheDocument();
  });

  it("renders win rates", () => {
    render(<ProfileCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getByText("80.0%")).toBeInTheDocument();
  });

  it("renders stat rows", () => {
    render(<ProfileCards profiles={profiles} percentileMode="p95" />);
    expect(screen.getAllByText("Avg Duration").length).toBe(2);
    expect(screen.getAllByText("Median").length).toBe(2);
  });
});
