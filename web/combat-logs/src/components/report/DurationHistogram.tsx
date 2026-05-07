import { useMemo } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import type { PercentileStats } from "../../types";
import { computeDurationHistogram } from "../../analysis/report";
import { formatDuration } from "../../analysis/format";

interface Props {
  stats: PercentileStats;
  percentileMode: "p95" | "p99";
  setPercentileMode: (mode: "p95" | "p99") => void;
}

export function DurationHistogram({ stats, percentileMode, setPercentileMode }: Props) {
  const histogram = useMemo(
    () => computeDurationHistogram(stats.values, 25),
    [stats.values]
  );

  const pValue = percentileMode === "p99" ? stats.p99 : stats.p95;

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-sm font-medium text-text-muted">Duration Distribution</h4>
        <div className="flex gap-1">
          <button
            onClick={() => setPercentileMode("p95")}
            className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
              percentileMode === "p95"
                ? "bg-accent text-white"
                : "bg-bg text-text-muted hover:text-text"
            }`}
          >
            P95
          </button>
          <button
            onClick={() => setPercentileMode("p99")}
            className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
              percentileMode === "p99"
                ? "bg-accent text-white"
                : "bg-bg text-text-muted hover:text-text"
            }`}
          >
            P99
          </button>
        </div>
      </div>

      <ResponsiveContainer width="100%" height={220}>
        <BarChart data={histogram} barCategoryGap={1}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
          <XAxis
            dataKey="rangeLabel"
            stroke="var(--color-text-muted)"
            fontSize={11}
            interval="preserveStartEnd"
          />
          <YAxis stroke="var(--color-text-muted)" fontSize={11} />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--color-surface)",
              border: "1px solid var(--color-border)",
              borderRadius: 4,
              color: "var(--color-text)",
              fontSize: 12,
            }}
            formatter={(value) => [`${value} runs`, "Count"]}
          />
          <ReferenceLine
            x={findClosestBucket(histogram, stats.median)}
            stroke="var(--color-accent)"
            strokeDasharray="4 4"
            label={{ value: "Median", position: "top", fill: "var(--color-accent)", fontSize: 10 }}
          />
          <ReferenceLine
            x={findClosestBucket(histogram, pValue)}
            stroke="var(--color-warning)"
            strokeDasharray="4 4"
            label={{
              value: `${percentileMode.toUpperCase()}: ${formatDuration(pValue)}`,
              position: "top",
              fill: "var(--color-warning)",
              fontSize: 10,
            }}
          />
          <Bar dataKey="count" fill="var(--color-accent)" opacity={0.7} radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function findClosestBucket(
  histogram: { rangeLabel: string; minMs: number; maxMs: number }[],
  valueMs: number
): string | undefined {
  for (const b of histogram) {
    if (valueMs >= b.minMs && valueMs < b.maxMs) return b.rangeLabel;
  }
  return histogram[histogram.length - 1]?.rangeLabel;
}
