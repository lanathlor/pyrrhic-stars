import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import type { ClassDPSDistribution } from "../../types";
import { CLASS_COLORS, CLASS_DISPLAY_NAMES } from "../../constants";
import { formatDps } from "../../analysis/format";

interface Props {
  distributions: ClassDPSDistribution[];
  percentileMode: "p95" | "p99";
  label?: string;
}

export function ClassDPSChart({ distributions, percentileMode, label = "DPS by Class" }: Props) {
  if (distributions.length === 0) return null;

  const data = distributions.map((d) => ({
    name: CLASS_DISPLAY_NAMES[d.className] ?? d.className,
    className: d.className,
    median: Math.round(d.stats.median),
    avg: Math.round(d.stats.avg),
    pValue: Math.round(percentileMode === "p99" ? d.stats.p99 : d.stats.p95),
    min: Math.round(d.stats.min),
    max: Math.round(d.stats.max),
  }));

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <h4 className="text-sm font-medium text-text-muted mb-3">{label}</h4>

      {/* Stat cards per class */}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-3 mb-4">
        {data.map((d) => {
          const color = CLASS_COLORS[d.className] ?? "var(--color-accent)";
          return (
            <div
              key={d.className}
              className="bg-bg rounded-lg p-3 border-l-4"
              style={{ borderLeftColor: color }}
            >
              <div className="text-xs text-text-muted mb-1">{d.name}</div>
              <div className="text-xl font-bold tabular-nums">{formatDps(d.median)}</div>
              <div className="text-xs text-text-muted mt-1 tabular-nums">
                avg {formatDps(d.avg)} · {percentileMode.toUpperCase()} {formatDps(d.pValue)}
              </div>
            </div>
          );
        })}
      </div>

      {/* Bar chart */}
      <ResponsiveContainer width="100%" height={50 + distributions.length * 44}>
        <BarChart data={data} layout="vertical" barSize={18} margin={{ left: 10, right: 30 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" horizontal={false} />
          <XAxis
            type="number"
            stroke="var(--color-text-muted)"
            fontSize={11}
            tickFormatter={(v) => formatDps(v)}
          />
          <YAxis
            type="category"
            dataKey="name"
            stroke="var(--color-text-muted)"
            fontSize={12}
            width={100}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--color-surface)",
              border: "1px solid var(--color-border)",
              borderRadius: 4,
              color: "var(--color-text)",
              fontSize: 12,
            }}
            formatter={(value) => [formatDps(Number(value)), "Median DPS"]}
          />
          {/* P95/P99 bars (behind median) */}
          <Bar dataKey="pValue" opacity={0.2} radius={[0, 4, 4, 0]}>
            {data.map((entry) => (
              <Cell key={entry.className} fill={CLASS_COLORS[entry.className] ?? "var(--color-accent)"} />
            ))}
          </Bar>
          {/* Median bars (on top) */}
          <Bar dataKey="median" radius={[0, 4, 4, 0]}>
            {data.map((entry) => (
              <Cell key={entry.className} fill={CLASS_COLORS[entry.className] ?? "var(--color-accent)"} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
      <div className="flex gap-4 mt-1 text-xs text-text-muted justify-center">
        <span>Solid = median</span>
        <span>Faded = {percentileMode.toUpperCase()}</span>
      </div>
    </div>
  );
}
