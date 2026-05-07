import { render, screen, fireEvent } from "@testing-library/react";
import { FightListFilters } from "./FightListFilters";

describe("FightListFilters", () => {
  it("renders encounter input", () => {
    render(
      <FightListFilters
        encounter="" setEncounter={() => {}}
        outcome="" setOutcome={() => {}}
        source="" setSource={() => {}}
      />
    );
    expect(screen.getByPlaceholderText("Encounter name...")).toBeInTheDocument();
  });

  it("renders outcome dropdown", () => {
    render(
      <FightListFilters
        encounter="" setEncounter={() => {}}
        outcome="" setOutcome={() => {}}
        source="" setSource={() => {}}
      />
    );
    expect(screen.getByText("All Outcomes")).toBeInTheDocument();
    expect(screen.getByText("Kill")).toBeInTheDocument();
    expect(screen.getByText("Wipe")).toBeInTheDocument();
  });

  it("renders source dropdown", () => {
    render(
      <FightListFilters
        encounter="" setEncounter={() => {}}
        outcome="" setOutcome={() => {}}
        source="" setSource={() => {}}
      />
    );
    expect(screen.getByText("All Sources")).toBeInTheDocument();
    expect(screen.getByText("Live")).toBeInTheDocument();
    expect(screen.getByText("Simulation")).toBeInTheDocument();
  });

  it("calls setEncounter on input change", () => {
    const setEncounter = vi.fn();
    render(
      <FightListFilters
        encounter="" setEncounter={setEncounter}
        outcome="" setOutcome={() => {}}
        source="" setSource={() => {}}
      />
    );
    fireEvent.change(screen.getByPlaceholderText("Encounter name..."), { target: { value: "boss" } });
    expect(setEncounter).toHaveBeenCalledWith("boss");
  });

  it("calls setOutcome on outcome select change", () => {
    const setOutcome = vi.fn();
    render(
      <FightListFilters
        encounter="" setEncounter={() => {}}
        outcome="" setOutcome={setOutcome}
        source="" setSource={() => {}}
      />
    );
    const selects = screen.getAllByRole("combobox");
    fireEvent.change(selects[0], { target: { value: "player_win" } });
    expect(setOutcome).toHaveBeenCalledWith("player_win");
  });

  it("calls setSource on source select change", () => {
    const setSource = vi.fn();
    render(
      <FightListFilters
        encounter="" setEncounter={() => {}}
        outcome="" setOutcome={() => {}}
        source="" setSource={setSource}
      />
    );
    const selects = screen.getAllByRole("combobox");
    fireEvent.change(selects[1], { target: { value: "simulation" } });
    expect(setSource).toHaveBeenCalledWith("simulation");
  });
});
