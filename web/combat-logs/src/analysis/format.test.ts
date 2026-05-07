import { formatDuration, formatTimestamp, formatAmount, formatDps, formatPercent, formatAbilityName } from "./format";

describe("formatDuration", () => {
  it("formats zero", () => {
    expect(formatDuration(0)).toBe("0:00");
  });

  it("formats seconds only", () => {
    expect(formatDuration(5000)).toBe("0:05");
    expect(formatDuration(45000)).toBe("0:45");
  });

  it("formats minutes and seconds", () => {
    expect(formatDuration(61000)).toBe("1:01");
    expect(formatDuration(125000)).toBe("2:05");
  });

  it("formats hours", () => {
    expect(formatDuration(3661000)).toBe("1:01:01");
  });
});

describe("formatTimestamp", () => {
  it("formats ms to seconds with one decimal", () => {
    expect(formatTimestamp(1500)).toBe("1.5s");
    expect(formatTimestamp(0)).toBe("0.0s");
  });
});

describe("formatAmount", () => {
  it("rounds and formats with locale separators", () => {
    expect(formatAmount(1234.7)).toMatch(/1.?235/);
    expect(formatAmount(0)).toBe("0");
  });
});

describe("formatDps", () => {
  it("returns raw number for small values", () => {
    expect(formatDps(500)).toBe("500");
  });

  it("formats thousands with K suffix", () => {
    expect(formatDps(1500)).toBe("1.5K");
    expect(formatDps(10000)).toBe("10.0K");
  });

  it("formats millions with M suffix", () => {
    expect(formatDps(2_500_000)).toBe("2.5M");
  });
});

describe("formatPercent", () => {
  it("converts decimal to percent string", () => {
    expect(formatPercent(0.5)).toBe("50.0%");
    expect(formatPercent(0)).toBe("0.0%");
    expect(formatPercent(1)).toBe("100.0%");
  });
});

describe("formatAbilityName", () => {
  it("returns Auto Attack for empty string", () => {
    expect(formatAbilityName("")).toBe("Auto Attack");
  });

  it("converts snake_case to Title Case", () => {
    expect(formatAbilityName("heavy_slash")).toBe("Heavy Slash");
    expect(formatAbilityName("ground_slam_aoe")).toBe("Ground Slam Aoe");
  });

  it("handles single word", () => {
    expect(formatAbilityName("fireball")).toBe("Fireball");
  });
});
