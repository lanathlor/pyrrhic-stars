import { render, screen, fireEvent } from "@testing-library/react";
import { DurationHistogram } from "./DurationHistogram";
import type { PercentileStats } from "../../types";

vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart">{children}</div>
    ),
  };
});

const stats: PercentileStats = {
  min: 20000, max: 80000, avg: 45000, median: 42000, p95: 70000, p99: 78000,
  values: Array.from({ length: 50 }, (_, i) => 20000 + i * 1200),
};

describe("DurationHistogram", () => {
  it("renders heading", () => {
    render(<DurationHistogram stats={stats} percentileMode="p95" setPercentileMode={() => {}} />);
    expect(screen.getByText("Duration Distribution")).toBeInTheDocument();
  });

  it("renders P95 and P99 toggle buttons", () => {
    render(<DurationHistogram stats={stats} percentileMode="p95" setPercentileMode={() => {}} />);
    expect(screen.getByText("P95")).toBeInTheDocument();
    expect(screen.getByText("P99")).toBeInTheDocument();
  });

  it("calls setPercentileMode on button click", () => {
    const setMode = vi.fn();
    render(<DurationHistogram stats={stats} percentileMode="p95" setPercentileMode={setMode} />);
    fireEvent.click(screen.getByText("P99"));
    expect(setMode).toHaveBeenCalledWith("p99");
  });

  it("renders chart container", () => {
    render(<DurationHistogram stats={stats} percentileMode="p95" setPercentileMode={() => {}} />);
    expect(screen.getByTestId("chart")).toBeInTheDocument();
  });
});
