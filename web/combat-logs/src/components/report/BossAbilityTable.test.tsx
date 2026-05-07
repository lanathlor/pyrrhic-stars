import { render, screen } from "@testing-library/react";
import { BossAbilityTable } from "./BossAbilityTable";
import type { BossAbilityStat } from "../../types";

const abilities: BossAbilityStat[] = [
  { ability_id: "cleave", total_damage: 50000, hits: 200, kills: 10, dodges: 40 },
  { ability_id: "ground_slam", total_damage: 80000, hits: 100, kills: 30, dodges: 20 },
];

describe("BossAbilityTable", () => {
  it("renders table headers", () => {
    render(<BossAbilityTable abilities={abilities} />);
    expect(screen.getByText("Ability")).toBeInTheDocument();
    expect(screen.getByText("Damage Share")).toBeInTheDocument();
    expect(screen.getByText("Total Damage")).toBeInTheDocument();
    expect(screen.getByText("Hits")).toBeInTheDocument();
    expect(screen.getByText("Kill Rate")).toBeInTheDocument();
    expect(screen.getByText("Dodge Rate")).toBeInTheDocument();
  });

  it("renders ability names", () => {
    render(<BossAbilityTable abilities={abilities} />);
    expect(screen.getByText("Cleave")).toBeInTheDocument();
    expect(screen.getByText("Ground Slam")).toBeInTheDocument();
  });

  it("returns null for empty abilities", () => {
    const { container } = render(<BossAbilityTable abilities={[]} />);
    expect(container.innerHTML).toBe("");
  });
});
