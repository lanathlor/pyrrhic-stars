import {
  Tooltip,
  ResponsiveContainer,
  Cell,
  PieChart,
  Pie,
  Legend,
} from "recharts";
import type { PhaseReachEntry, WipePhaseEntry } from "../../types";
import { formatPercent } from "../../analysis/format";

interface Props {
  phaseReach: PhaseReachEntry[];
  wipePhases: WipePhaseEntry[];
}

const PHASE_COLORS = [
  "var(--color-success)",
  "var(--color-accent)",
  "var(--color-warning)",
  "var(--color-danger)",
  "#e879f9",
  "#fb923c",
];

function formatPhaseName(phase: string): string {
  return phase
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function PhaseAnalysis({ phaseReach, wipePhases }: Props) {
  if (phaseReach.length === 0) return null;

  const reachData = phaseReach.map((p, i) => ({
    name: formatPhaseName(p.phase),
    rate: Math.round(p.rate * 1000) / 10,
    count: p.count,
    fill: PHASE_COLORS[i % PHASE_COLORS.length],
  }));

  const totalWipes = wipePhases.reduce((s, w) => s + w.count, 0);
  const wipeData = wipePhases.map((w, i) => ({
    name: formatPhaseName(w.phase),
    value: w.count,
    fill: PHASE_COLORS[i % PHASE_COLORS.length],
  }));

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <h4 className="text-sm font-medium text-text-muted mb-3">Phase Analysis</h4>

      {/* Phase reach rates */}
      <div className="mb-4">
        <div className="text-xs text-text-muted mb-2">Phase Reach Rates</div>
        <div className="space-y-2">
          {reachData.map((d) => (
            <div key={d.name}>
              <div className="flex items-center justify-between text-sm mb-0.5">
                <span>{d.name}</span>
                <span className="tabular-nums">{d.rate}%</span>
              </div>
              <div className="h-2 bg-bg rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full transition-all"
                  style={{ width: `${d.rate}%`, backgroundColor: d.fill }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Wipe phase distribution */}
      {wipeData.length > 0 && (
        <div>
          <div className="text-xs text-text-muted mb-2">Wipe Distribution ({totalWipes} wipes)</div>
          <ResponsiveContainer width="100%" height={160}>
            <PieChart>
              <Pie
                data={wipeData}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                innerRadius={35}
                outerRadius={60}
                strokeWidth={2}
                stroke="var(--color-bg)"
              >
                {wipeData.map((entry, i) => (
                  <Cell key={i} fill={entry.fill} />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{
                  backgroundColor: "var(--color-surface)",
                  border: "1px solid var(--color-border)",
                  borderRadius: 4,
                  color: "var(--color-text)",
                  fontSize: 12,
                }}
                formatter={(value) => [
                  `${value} (${formatPercent(Number(value) / totalWipes)})`,
                  "Wipes",
                ]}
              />
              <Legend
                formatter={(value) => <span className="text-xs text-text">{value}</span>}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
