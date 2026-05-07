import { useState } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { formatAmount, formatDps, formatPercent, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";

export function HealingTab() {
  const { analysis } = useFightContext();
  const { healingDone } = analysis;
  const [expanded, setExpanded] = useState<string | null>(null);

  if (healingDone.length === 0) {
    return (
      <div className="min-h-[200px]">
        <p className="text-text-muted">No healing events recorded.</p>
      </div>
    );
  }

  return (
    <div className="min-h-[200px]">
      <table className="w-full border-collapse">
        <thead>
          <tr>
            <th>Player</th>
            <th className="text-right">Total</th>
            <th className="text-right">HPS</th>
            <th className="text-right">Crit%</th>
            <th className="text-right">Hits</th>
          </tr>
        </thead>
        <tbody>
          {healingDone.map((h) => (
            <>
              <tr
                key={h.entityId}
                onClick={() => setExpanded(expanded === h.entityId ? null : h.entityId)}
                className={`cursor-pointer ${expanded === h.entityId ? "!bg-surface" : ""}`}
              >
                <td>
                  <ClassIcon className={h.className} />
                  <span className="ml-2">{h.name}</span>
                </td>
                <td className="text-right">{formatAmount(h.totalHealing)}</td>
                <td className="text-right">{formatDps(h.hps)}</td>
                <td className="text-right">{formatPercent(h.critRate)}</td>
                <td className="text-right">{h.hitCount}</td>
              </tr>
              {expanded === h.entityId &&
                h.abilities.map((a) => (
                  <tr key={`${h.entityId}-${a.abilityId}`} className="bg-surface text-text-muted text-[0.8rem] [&>td]:border-b-border/50">
                    <td className="pl-8">{formatAbilityName(a.abilityId)}</td>
                    <td className="text-right">{formatAmount(a.totalDamage)}</td>
                    <td className="text-right">—</td>
                    <td className="text-right">{formatPercent(a.critRate)}</td>
                    <td className="text-right">{a.hitCount}</td>
                  </tr>
                ))}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}
