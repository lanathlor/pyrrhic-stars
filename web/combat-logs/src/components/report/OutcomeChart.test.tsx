import { render, screen } from "@testing-library/react";
import { OutcomeChart } from "./OutcomeChart";

vi.mock("recharts", async () => {
  const actual = await vi.importActual("recharts");
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="chart">{children}</div>
    ),
  };
});

describe("OutcomeChart", () => {
  it("renders outcome distribution heading", () => {
    render(<OutcomeChart wins={50} losses={30} timeouts={20} />);
    expect(screen.getByText("Outcome Distribution")).toBeInTheDocument();
  });

  it("renders chart container", () => {
    render(<OutcomeChart wins={50} losses={30} timeouts={0} />);
    expect(screen.getByTestId("chart")).toBeInTheDocument();
  });
});
