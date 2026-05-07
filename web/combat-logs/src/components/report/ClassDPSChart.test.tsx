import { render, screen } from "@testing-library/react";
import { ClassDPSChart } from "./ClassDPSChart";
import type { ClassDPSDistribution } from "../../types";

vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart">{children}</div>
    ),
  };
});

const distributions: ClassDPSDistribution[] = [
  { className: "gunner", stats: { min: 200, max: 800, avg: 500, median: 480, p95: 750, p99: 790, values: [200, 480, 800] } },
  { className: "vanguard", stats: { min: 100, max: 600, avg: 350, median: 330, p95: 550, p99: 590, values: [100, 330, 600] } },
];

describe("ClassDPSChart", () => {
  it("renders label heading", () => {
    render(<ClassDPSChart distributions={distributions} percentileMode="p95" label="DPS by Class" />);
    expect(screen.getByText("DPS by Class")).toBeInTheDocument();
  });

  it("renders class stat cards", () => {
    render(<ClassDPSChart distributions={distributions} percentileMode="p95" />);
    expect(screen.getByText("Gunner")).toBeInTheDocument();
    expect(screen.getByText("Vanguard")).toBeInTheDocument();
  });

  it("returns null for empty distributions", () => {
    const { container } = render(<ClassDPSChart distributions={[]} percentileMode="p95" />);
    expect(container.innerHTML).toBe("");
  });

  it("renders chart container", () => {
    render(<ClassDPSChart distributions={distributions} percentileMode="p95" />);
    expect(screen.getByTestId("chart")).toBeInTheDocument();
  });

  it("shows percentile legend", () => {
    render(<ClassDPSChart distributions={distributions} percentileMode="p95" />);
    expect(screen.getByText("Solid = median")).toBeInTheDocument();
    expect(screen.getByText(/Faded = P95/)).toBeInTheDocument();
  });
});
