import type { BossAbilityStat } from "../../types";
import { formatDps, formatPercent } from "../../analysis/format";
import { formatAbilityName } from "../../analysis/format";

interface Props {
  abilities: BossAbilityStat[];
}

export function BossAbilityTable({ abilities }: Props) {
  if (abilities.length === 0) return null;

  const totalDamage = abilities.reduce((s, a) => s + a.total_damage, 0);

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <h4 className="text-sm font-medium text-text-muted mb-3">Boss Abilities</h4>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th>Ability</th>
              <th className="text-right">Damage Share</th>
              <th className="text-right">Total Damage</th>
              <th className="text-right">Hits</th>
              <th className="text-right">Kill Rate</th>
              <th className="text-right">Dodge Rate</th>
            </tr>
          </thead>
          <tbody>
            {abilities.map((a) => {
              const share = totalDamage > 0 ? a.total_damage / totalDamage : 0;
              const totalInteractions = a.hits + a.dodges;
              const killRate = totalInteractions > 0 ? a.kills / totalInteractions : 0;
              const dodgeRate = totalInteractions > 0 ? a.dodges / totalInteractions : 0;

              return (
                <tr key={a.ability_id}>
                  <td>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{formatAbilityName(a.ability_id)}</span>
                    </div>
                    {/* Damage share bar */}
                    <div className="mt-1 h-1.5 bg-bg rounded-full overflow-hidden w-full max-w-[200px]">
                      <div
                        className="h-full rounded-full"
                        style={{
                          width: `${share * 100}%`,
                          backgroundColor: killRate > 0.15 ? "var(--color-danger)" : "var(--color-accent)",
                        }}
                      />
                    </div>
                  </td>
                  <td className="text-right tabular-nums font-medium">
                    {formatPercent(share)}
                  </td>
                  <td className="text-right tabular-nums">
                    {formatDps(a.total_damage)}
                  </td>
                  <td className="text-right tabular-nums">
                    {a.hits.toLocaleString()}
                  </td>
                  <td className="text-right tabular-nums">
                    <span className={killRate > 0.15 ? "text-danger" : ""}>
                      {formatPercent(killRate)}
                    </span>
                  </td>
                  <td className="text-right tabular-nums">
                    <span className={dodgeRate < 0.2 ? "text-warning" : "text-success"}>
                      {formatPercent(dodgeRate)}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
