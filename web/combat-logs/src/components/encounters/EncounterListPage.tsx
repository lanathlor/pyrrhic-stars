import { useMemo } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { fetchInstances } from "../../api";
import { groupByEncounter } from "../../analysis/report";
import { formatDuration, formatPercent } from "../../analysis/format";
import { ClassIcon } from "../shared/ClassIcon";

const PROFILE_LABELS: Record<string, string> = {
  sweaty: "Sweaty",
  average: "Average",
  bad: "Bad",
};

export function EncounterListPage() {
  const { data: instances = [], isLoading, error } = useQuery({
    queryKey: ["instances", { source: "simulation", limit: "10000" }],
    queryFn: () => fetchInstances({ source: "simulation", limit: "10000" }),
  });

  const encounters = useMemo(() => groupByEncounter(instances), [instances]);

  if (isLoading) return <p className="text-text-muted">Loading encounters...</p>;
  if (error) return <p className="text-danger">{error.message}</p>;

  if (encounters.length === 0) {
    return (
      <div>
        <p className="text-text-muted mb-4">No simulation data found.</p>
        <Link to="/fights" className="text-accent hover:underline text-sm">
          View live combat logs →
        </Link>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <p className="text-text-muted text-sm">
          {encounters.length} encounter{encounters.length !== 1 ? "s" : ""} · {instances.length} total runs
        </p>
        <Link to="/fights" className="text-accent hover:underline text-sm">
          View live logs →
        </Link>
      </div>

      <div className="grid gap-4">
        {encounters.map((enc) => (
          <Link
            key={enc.encounterId}
            to="/report/$encounterId"
            params={{ encounterId: enc.encounterId }}
            className="block bg-surface border border-border rounded-lg p-5 hover:border-accent transition-colors"
          >
            <div className="flex items-start justify-between mb-3">
              <div>
                <h2 className="text-lg font-semibold capitalize">
                  {enc.encounterId.replace(/_/g, " ")}
                </h2>
                <span className="text-text-muted text-sm">
                  {enc.totalRuns} runs · avg {formatDuration(enc.avgDurationMs)}
                </span>
              </div>
              <WinRateBadge rate={enc.winRate} />
            </div>

            <div className="flex gap-6 text-sm">
              <div className="flex items-center gap-4">
                <span className="text-success">{enc.wins} wins</span>
                <span className="text-danger">{enc.losses} losses</span>
                {enc.timeouts > 0 && <span className="text-warning">{enc.timeouts} timeouts</span>}
              </div>
              <div className="flex items-center gap-2 ml-auto">
                {enc.classes.map((cls) => (
                  <ClassIcon key={cls} className={cls} showName={false} />
                ))}
              </div>
              {enc.profiles.length > 0 && (
                <div className="flex gap-1.5">
                  {enc.profiles.map((p) => (
                    <span key={p} className="px-2 py-0.5 bg-bg rounded text-xs text-text-muted">
                      {PROFILE_LABELS[p] ?? p}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}

function WinRateBadge({ rate }: { rate: number }) {
  const color = rate >= 0.7 ? "text-success" : rate >= 0.3 ? "text-warning" : "text-danger";
  return (
    <span className={`text-2xl font-bold tabular-nums ${color}`}>
      {formatPercent(rate)}
    </span>
  );
}
