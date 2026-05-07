import { render, screen } from "@testing-library/react";
import { KPICard } from "./KPICard";

describe("KPICard", () => {
  it("renders label and value", () => {
    render(<KPICard label="Total Damage" value="1,234" />);
    expect(screen.getByText("Total Damage")).toBeInTheDocument();
    expect(screen.getByText("1,234")).toBeInTheDocument();
  });

  it("renders subtitle when provided", () => {
    render(<KPICard label="DPS" value="500" subtitle="per second" />);
    expect(screen.getByText("per second")).toBeInTheDocument();
  });

  it("does not render subtitle when absent", () => {
    const { container } = render(<KPICard label="Deaths" value="0" />);
    const spans = container.querySelectorAll("span");
    expect(spans).toHaveLength(2); // label + value only
  });
});
