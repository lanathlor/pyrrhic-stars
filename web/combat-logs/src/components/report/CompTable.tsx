import type { CompStats } from "../../types";
import { formatDuration, formatPercent } from "../../analysis/format";
import { ClassIcon } from "../shared/ClassIcon";

interface Props {
  comps: CompStats[];
  percentileMode: "p95" | "p99";
}

export function CompTable({ comps, percentileMode }: Props) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr>
            <th>Composition</th>
            <th>Profiles</th>
            <th className="text-right">Runs</th>
            <th className="text-right">Win%</th>
            <th className="text-right">Avg Duration</th>
            <th className="text-right">Median</th>
            <th className="text-right">{percentileMode.toUpperCase()}</th>
          </tr>
        </thead>
        <tbody>
          {comps.map((c) => {
            const pValue = percentileMode === "p99" ? c.durationStats.p99 : c.durationStats.p95;
            const winColor = c.winRate >= 0.7 ? "text-success" : c.winRate >= 0.3 ? "text-warning" : "text-danger";
            return (
              <tr key={c.name}>
                <td>
                  <div className="flex items-center gap-1.5">
                    {c.classes.map((cls, i) => (
                      <ClassIcon key={`${cls}-${i}`} className={cls} />
                    ))}
                  </div>
                </td>
                <td>
                  <div className="flex gap-1">
                    {c.profiles.map((p, i) => (
                      <span key={i} className="px-1.5 py-0.5 bg-bg rounded text-xs text-text-muted">
                        {p}
                      </span>
                    ))}
                  </div>
                </td>
                <td className="text-right tabular-nums">{c.runs}</td>
                <td className={`text-right tabular-nums font-medium ${winColor}`}>
                  {formatPercent(c.winRate)}
                </td>
                <td className="text-right tabular-nums">{formatDuration(c.avgDurationMs)}</td>
                <td className="text-right tabular-nums">{formatDuration(c.durationStats.median)}</td>
                <td className="text-right tabular-nums">{formatDuration(pValue)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
