import type { ProfileCombatStats } from "../../types";
import { formatDuration, formatPercent, formatDps } from "../../analysis/format";
import { CLASS_COLORS, CLASS_DISPLAY_NAMES } from "../../constants";

interface Props {
  profiles: ProfileCombatStats[];
  percentileMode: "p95" | "p99";
}

const PROFILE_STYLES: Record<string, { label: string; ring: string }> = {
  sweaty: { label: "Sweaty", ring: "border-accent" },
  average: { label: "Average", ring: "border-warning" },
  bad: { label: "Bad", ring: "border-danger" },
};

export function ProfileCombatCards({ profiles, percentileMode }: Props) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4">
      {profiles.map((p) => {
        const style = PROFILE_STYLES[p.profile] ?? { label: p.profile, ring: "border-border" };
        const durPValue = percentileMode === "p99" ? p.durationStats.p99 : p.durationStats.p95;
        const winColor = p.winRate >= 0.7 ? "text-success" : p.winRate >= 0.3 ? "text-warning" : "text-danger";

        return (
          <div
            key={p.profile}
            className={`bg-surface border-2 ${style.ring} rounded-lg p-4`}
          >
            {/* Header */}
            <div className="flex items-center justify-between mb-3">
              <h4 className="font-semibold">{style.label}</h4>
              <span className="text-text-muted text-sm">{p.runs} runs</span>
            </div>

            {/* Win rate */}
            <div className={`text-3xl font-bold tabular-nums mb-3 ${winColor}`}>
              {formatPercent(p.winRate)}
            </div>

            {/* Duration stats */}
            <div className="grid grid-cols-2 gap-y-1.5 text-sm mb-4">
              <StatRow label="Avg Duration" value={formatDuration(p.avgDurationMs)} />
              <StatRow label="Median" value={formatDuration(p.durationStats.median)} />
              <StatRow label={percentileMode.toUpperCase()} value={formatDuration(durPValue)} />
              <StatRow
                label="Outcomes"
                value={`${p.wins}W/${p.losses}L${p.timeouts > 0 ? `/${p.timeouts}T` : ""}`}
              />
            </div>

            {/* Class DPS breakdown */}
            {p.classDPS.length > 0 && (
              <div className="border-t border-border pt-3 mb-3">
                <div className="text-xs text-text-muted mb-2">Median DPS by Class</div>
                <div className="space-y-1.5">
                  {p.classDPS.map((c) => {
                    const color = CLASS_COLORS[c.className] ?? "var(--color-accent)";
                    const pDps = percentileMode === "p99" ? c.stats.p99 : c.stats.p95;
                    return (
                      <div key={c.className} className="flex items-center gap-2">
                        <div
                          className="w-2 h-2 rounded-full shrink-0"
                          style={{ backgroundColor: color }}
                        />
                        <span className="text-sm flex-1">
                          {CLASS_DISPLAY_NAMES[c.className] ?? c.className}
                        </span>
                        <span className="text-sm tabular-nums font-medium">
                          {formatDps(c.stats.median)}
                        </span>
                        <span className="text-xs text-text-muted tabular-nums">
                          {percentileMode.toUpperCase()} {formatDps(pDps)}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Deaths */}
            <div className="border-t border-border pt-3">
              <div className="grid grid-cols-2 gap-y-1.5 text-sm">
                <StatRow label="Deaths/Run (med)" value={p.deathStats.median.toFixed(1)} />
                <StatRow label="Deaths/Run (avg)" value={p.deathStats.avg.toFixed(1)} />
              </div>
            </div>

            {/* Phase reach */}
            {Object.keys(p.phaseReach).length > 1 && (
              <div className="border-t border-border pt-3 mt-3">
                <div className="text-xs text-text-muted mb-1.5">Phase Reach</div>
                <div className="flex gap-2 flex-wrap">
                  {Object.entries(p.phaseReach)
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([phase, rate]) => (
                      <span
                        key={phase}
                        className="px-2 py-0.5 bg-bg rounded text-xs tabular-nums"
                      >
                        {phase.replace("phase_", "P")}: {(rate * 100).toFixed(0)}%
                      </span>
                    ))}
                </div>
              </div>
            )}
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
