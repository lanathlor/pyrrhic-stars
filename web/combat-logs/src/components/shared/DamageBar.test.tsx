import { render, screen } from "@testing-library/react";
import { DamageBar } from "./DamageBar";

describe("DamageBar", () => {
  it("renders name and value", () => {
    render(<DamageBar name="Alice" className="gunner" value="1,234" percent={0.8} />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("1,234")).toBeInTheDocument();
  });

  it("renders secondary text when provided", () => {
    render(<DamageBar name="Alice" className="gunner" value="1,234" secondary="500 DPS" percent={0.5} />);
    expect(screen.getByText("500 DPS")).toBeInTheDocument();
  });

  it("does not render secondary when absent", () => {
    const { container } = render(<DamageBar name="Alice" className="gunner" value="1,234" percent={0.5} />);
    expect(container.querySelectorAll(".text-text-muted")).toHaveLength(0);
  });

  it("sets bar width based on percent", () => {
    const { container } = render(<DamageBar name="Alice" className="gunner" value="100" percent={0.6} />);
    const bar = container.querySelector("[style]");
    expect(bar).toHaveStyle({ width: "60%" });
  });

  it("enforces minimum 2% width", () => {
    const { container } = render(<DamageBar name="Alice" className="gunner" value="0" percent={0} />);
    const bar = container.querySelector("[style]");
    expect(bar).toHaveStyle({ width: "2%" });
  });
});
