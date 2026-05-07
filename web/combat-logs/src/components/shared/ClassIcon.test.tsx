import { render, screen } from "@testing-library/react";
import { ClassIcon } from "./ClassIcon";

describe("ClassIcon", () => {
  it("renders display name for known class", () => {
    render(<ClassIcon className="gunner" />);
    expect(screen.getByText("Gunner")).toBeInTheDocument();
  });

  it("renders raw class name for unknown class", () => {
    render(<ClassIcon className="mystery" />);
    expect(screen.getByText("mystery")).toBeInTheDocument();
  });

  it("hides name when showName is false", () => {
    const { container } = render(<ClassIcon className="gunner" showName={false} />);
    expect(container.textContent).toBe("");
    // Outer wrapper span + dot span, but no text span
    expect(container.querySelectorAll("span")).toHaveLength(2);
  });

  it("renders a colored dot", () => {
    const { container } = render(<ClassIcon className="vanguard" />);
    const dot = container.querySelector("span span");
    expect(dot).toHaveStyle({ backgroundColor: "#3b82f6" });
  });
});
