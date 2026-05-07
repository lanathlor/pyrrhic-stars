import type { ProfileStats } from "../../types";
import { formatDuration, formatPercent } from "../../analysis/format";

interface Props {
  profiles: ProfileStats[];
  percentileMode: "p95" | "p99";
}

const PROFILE_STYLES: Record<string, { label: string; ring: string }> = {
  sweaty: { label: "Sweaty", ring: "border-accent" },
  average: { label: "Average", ring: "border-warning" },
  bad: { label: "Bad", ring: "border-danger" },
};

export function ProfileCards({ profiles, percentileMode }: Props) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-4">
      {profiles.map((p) => {
        const style = PROFILE_STYLES[p.profile] ?? { label: p.profile, ring: "border-border" };
        const pValue = percentileMode === "p99" ? p.durationStats.p99 : p.durationStats.p95;
        const winColor = p.winRate >= 0.7 ? "text-success" : p.winRate >= 0.3 ? "text-warning" : "text-danger";

        return (
          <div
            key={p.profile}
            className={`bg-surface border-2 ${style.ring} rounded-lg p-4`}
          >
            <div className="flex items-center justify-between mb-3">
              <h4 className="font-semibold">{style.label}</h4>
              <span className="text-text-muted text-sm">{p.runs} runs</span>
            </div>

            <div className={`text-3xl font-bold tabular-nums mb-3 ${winColor}`}>
              {formatPercent(p.winRate)}
            </div>

            <div className="grid grid-cols-2 gap-y-2 text-sm">
              <StatRow label="Avg Duration" value={formatDuration(p.avgDurationMs)} />
              <StatRow label="Median" value={formatDuration(p.durationStats.median)} />
              <StatRow label={percentileMode.toUpperCase()} value={formatDuration(pValue)} />
              <StatRow
                label="Outcomes"
                value={`${p.wins}W/${p.losses}L${p.timeouts > 0 ? `/${p.timeouts}T` : ""}`}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function StatRow({ label, value }: { label: string; value: string }) {
  return (
    <>
      <span className="text-text-muted">{label}</span>
      <span className="text-right tabular-nums">{value}</span>
    </>
  );
}
