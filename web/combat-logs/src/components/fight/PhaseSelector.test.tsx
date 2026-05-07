import { render, screen, fireEvent } from "@testing-library/react";
import { PhaseSelector } from "./PhaseSelector";
import type { PhaseMarker } from "../../types";

const phases: PhaseMarker[] = [
  { phase: "phase_1", startMs: 0, endMs: 5000 },
  { phase: "phase_2", startMs: 5000, endMs: 10000 },
];

describe("PhaseSelector", () => {
  it("renders All Phases option", () => {
    render(<PhaseSelector phases={phases} selected={null} onSelect={() => {}} />);
    expect(screen.getByText("All Phases")).toBeInTheDocument();
  });

  it("renders phase options", () => {
    render(<PhaseSelector phases={phases} selected={null} onSelect={() => {}} />);
    expect(screen.getByText("phase_1")).toBeInTheDocument();
    expect(screen.getByText("phase_2")).toBeInTheDocument();
  });

  it("calls onSelect with phase value", () => {
    const onSelect = vi.fn();
    render(<PhaseSelector phases={phases} selected={null} onSelect={onSelect} />);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "phase_1" } });
    expect(onSelect).toHaveBeenCalledWith("phase_1");
  });

  it("calls onSelect with null for All Phases", () => {
    const onSelect = vi.fn();
    render(<PhaseSelector phases={phases} selected="phase_1" onSelect={onSelect} />);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "" } });
    expect(onSelect).toHaveBeenCalledWith(null);
  });
});
