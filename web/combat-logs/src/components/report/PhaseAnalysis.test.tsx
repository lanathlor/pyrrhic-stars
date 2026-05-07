import { render, screen } from "@testing-library/react";
import { PhaseAnalysis } from "./PhaseAnalysis";
import type { PhaseReachEntry, WipePhaseEntry } from "../../types";

vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart">{children}</div>
    ),
  };
});

const phaseReach: PhaseReachEntry[] = [
  { phase: "phase_1", rate: 1.0, count: 100 },
  { phase: "phase_2", rate: 0.7, count: 70 },
  { phase: "phase_3", rate: 0.3, count: 30 },
];

const wipePhases: WipePhaseEntry[] = [
  { phase: "phase_1", count: 10 },
  { phase: "phase_2", count: 40 },
  { phase: "phase_3", count: 20 },
];

describe("PhaseAnalysis", () => {
  it("renders heading", () => {
    render(<PhaseAnalysis phaseReach={phaseReach} wipePhases={wipePhases} />);
    expect(screen.getByText("Phase Analysis")).toBeInTheDocument();
  });

  it("renders phase reach rates", () => {
    render(<PhaseAnalysis phaseReach={phaseReach} wipePhases={wipePhases} />);
    expect(screen.getByText("Phase Reach Rates")).toBeInTheDocument();
    expect(screen.getByText("Phase 1")).toBeInTheDocument();
    expect(screen.getByText("Phase 2")).toBeInTheDocument();
    expect(screen.getByText("Phase 3")).toBeInTheDocument();
    expect(screen.getByText("100%")).toBeInTheDocument();
    expect(screen.getByText("70%")).toBeInTheDocument();
    expect(screen.getByText("30%")).toBeInTheDocument();
  });

  it("renders wipe distribution", () => {
    render(<PhaseAnalysis phaseReach={phaseReach} wipePhases={wipePhases} />);
    expect(screen.getByText(/Wipe Distribution \(70 wipes\)/)).toBeInTheDocument();
  });

  it("returns null for empty phase reach", () => {
    const { container } = render(<PhaseAnalysis phaseReach={[]} wipePhases={[]} />);
    expect(container.innerHTML).toBe("");
  });
});
