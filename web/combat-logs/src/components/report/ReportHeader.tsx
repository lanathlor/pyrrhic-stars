import type { ReportOverview, PercentileStats } from "../../types";
import { formatDuration, formatPercent, formatDps } from "../../analysis/format";
import { KPICard } from "../shared/KPICard";

interface Props {
  overview: ReportOverview;
  raidDPSStats?: PercentileStats | null;
  deathStats?: PercentileStats | null;
  percentileMode: "p95" | "p99";
}

export function ReportHeader({ overview, raidDPSStats, deathStats, percentileMode }: Props) {
  const d = overview.durationStats;
  const durP = percentileMode === "p99" ? d.p99 : d.p95;

  return (
    <div className="mb-6">
      <h2 className="text-2xl font-bold capitalize mb-1">
        {overview.encounterId.replace(/_/g, " ")}
      </h2>
      <p className="text-text-muted text-sm mb-4">
        {overview.totalRuns} simulation runs ·{" "}
        {new Date(overview.firstRun).toLocaleDateString()} – {new Date(overview.lastRun).toLocaleDateString()}
      </p>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(140px,1fr))] gap-3 mb-6">
        <KPICard label="Win Rate" value={formatPercent(overview.winRate)} />
        <KPICard
          label="Total Runs"
          value={overview.totalRuns.toString()}
          subtitle={`${overview.wins}W / ${overview.losses}L / ${overview.timeouts}T`}
        />
        <KPICard label="Avg Duration" value={formatDuration(d.avg)} />
        <KPICard
          label={`Duration ${percentileMode.toUpperCase()}`}
          value={formatDuration(durP)}
          subtitle={`median ${formatDuration(d.median)}`}
        />
        {raidDPSStats && (
          <KPICard
            label="Raid DPS (med)"
            value={formatDps(raidDPSStats.median)}
            subtitle={`${percentileMode.toUpperCase()} ${formatDps(
              percentileMode === "p99" ? raidDPSStats.p99 : raidDPSStats.p95
            )}`}
          />
        )}
        {deathStats && (
          <KPICard
            label="Deaths/Run"
            value={deathStats.median.toFixed(1)}
            subtitle={`avg ${deathStats.avg.toFixed(1)}`}
          />
        )}
      </div>
    </div>
  );
}
