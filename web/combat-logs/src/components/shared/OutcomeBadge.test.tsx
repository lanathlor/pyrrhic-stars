import { render, screen } from "@testing-library/react";
import { OutcomeBadge } from "./OutcomeBadge";

describe("OutcomeBadge", () => {
  it("renders player_win with success class", () => {
    const { container } = render(<OutcomeBadge outcome="player_win" />);
    expect(screen.getByText("player win")).toBeInTheDocument();
    expect(container.querySelector("span")).toHaveClass("text-success");
  });

  it("renders boss_win with danger class", () => {
    const { container } = render(<OutcomeBadge outcome="boss_win" />);
    expect(screen.getByText("boss win")).toBeInTheDocument();
    expect(container.querySelector("span")).toHaveClass("text-danger");
  });

  it("renders unknown outcome with warning class", () => {
    const { container } = render(<OutcomeBadge outcome="timeout" />);
    expect(screen.getByText("timeout")).toBeInTheDocument();
    expect(container.querySelector("span")).toHaveClass("text-warning");
  });
});
