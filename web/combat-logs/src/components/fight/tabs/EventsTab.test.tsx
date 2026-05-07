import { screen } from "@testing-library/react";
import { EventsTab, eventRowClass } from "./EventsTab";
import { renderWithFight } from "../../../test/renderWithFight";
import { EVENT_TYPES } from "../../../constants";

describe("eventRowClass", () => {
  it("returns danger class for damage", () => {
    expect(eventRowClass(EVENT_TYPES.DAMAGE)).toContain("danger");
  });

  it("returns success class for heal", () => {
    expect(eventRowClass(EVENT_TYPES.HEAL)).toContain("success");
  });

  it("returns warning class for death", () => {
    expect(eventRowClass(EVENT_TYPES.DEATH)).toContain("warning");
  });

  it("returns empty for other types", () => {
    expect(eventRowClass(EVENT_TYPES.BUFF_APPLY)).toBe("");
  });
});

describe("EventsTab", () => {
  it("renders event table headers", () => {
    renderWithFight(<EventsTab />);
    expect(screen.getByText("Time")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Source")).toBeInTheDocument();
    expect(screen.getByText("Target")).toBeInTheDocument();
    expect(screen.getByText("Ability")).toBeInTheDocument();
    expect(screen.getByText("Amount")).toBeInTheDocument();
    expect(screen.getByText("Phase")).toBeInTheDocument();
  });

  it("shows event count", () => {
    renderWithFight(<EventsTab />);
    expect(screen.getByText(/\d+ events/)).toBeInTheDocument();
  });

  it("renders event type filter checkboxes", () => {
    renderWithFight(<EventsTab />);
    // Checkboxes are rendered as labels with checkbox input
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes.length).toBeGreaterThanOrEqual(3);
  });

  it("renders source and target filter selects", () => {
    renderWithFight(<EventsTab />);
    expect(screen.getByText("All Sources")).toBeInTheDocument();
    expect(screen.getByText("All Targets")).toBeInTheDocument();
  });
});
