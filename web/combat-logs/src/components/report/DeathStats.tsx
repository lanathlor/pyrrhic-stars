import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import type { PercentileStats, BossAbilityStat } from "../../types";
import { formatAbilityName } from "../../analysis/format";

interface Props {
  deathStats: PercentileStats;
  bossAbilities: BossAbilityStat[];
  percentileMode: "p95" | "p99";
}

export function DeathStats({ deathStats, bossAbilities, percentileMode }: Props) {
  // Build histogram of deaths per run (reuse duration histogram with integer values)
  const histogram = buildDeathHistogram(deathStats.values);
  const pValue = percentileMode === "p99" ? deathStats.p99 : deathStats.p95;

  // Top killers
  const killers = bossAbilities
    .filter((a) => a.kills > 0)
    .sort((a, b) => b.kills - a.kills)
    .slice(0, 5);

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <h4 className="text-sm font-medium text-text-muted mb-3">Deaths per Run</h4>

      <div className="grid grid-cols-3 gap-3 mb-4 text-center">
        <div>
          <div className="text-2xl font-bold tabular-nums">{deathStats.median.toFixed(1)}</div>
          <div className="text-xs text-text-muted">Median</div>
        </div>
        <div>
          <div className="text-2xl font-bold tabular-nums">{deathStats.avg.toFixed(1)}</div>
          <div className="text-xs text-text-muted">Avg</div>
        </div>
        <div>
          <div className="text-2xl font-bold tabular-nums">{Math.round(pValue)}</div>
          <div className="text-xs text-text-muted">{percentileMode.toUpperCase()}</div>
        </div>
      </div>

      <ResponsiveContainer width="100%" height={140}>
        <BarChart data={histogram} barCategoryGap={1}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
          <XAxis dataKey="label" stroke="var(--color-text-muted)" fontSize={11} />
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
          <Bar dataKey="count" fill="var(--color-danger)" opacity={0.7} radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>

      {killers.length > 0 && (
        <div className="mt-4">
          <div className="text-xs text-text-muted mb-2">Most Lethal Abilities</div>
          <div className="space-y-1.5">
            {killers.map((a) => {
              const total = a.hits + a.dodges;
              const killRate = total > 0 ? a.kills / total : 0;
              return (
                <div key={a.ability_id} className="flex items-center justify-between text-sm">
                  <span>{formatAbilityName(a.ability_id)}</span>
                  <span className="text-danger tabular-nums">
                    {a.kills} kills ({(killRate * 100).toFixed(0)}%)
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function buildDeathHistogram(values: number[]): { label: string; count: number }[] {
  if (values.length === 0) return [];
  const max = values[values.length - 1];
  const buckets: { label: string; count: number }[] = [];
  for (let i = 0; i <= max; i++) {
    buckets.push({ label: String(i), count: 0 });
  }
  for (const v of values) {
    const idx = Math.min(Math.round(v), max);
    if (buckets[idx]) buckets[idx].count++;
  }
  return buckets;
}
