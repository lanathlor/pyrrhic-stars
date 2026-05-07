import { useState } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { formatAmount, formatDps, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";

export function DamageTakenTab() {
  const { analysis } = useFightContext();
  const { damageTaken } = analysis;
  const [expanded, setExpanded] = useState<string | null>(null);

  return (
    <div className="min-h-[200px]">
      <table className="w-full border-collapse">
        <thead>
          <tr>
            <th>Player</th>
            <th className="text-right">Total Taken</th>
            <th className="text-right">DTPS</th>
          </tr>
        </thead>
        <tbody>
          {damageTaken.map((d) => (
            <>
              <tr
                key={d.entityId}
                onClick={() => setExpanded(expanded === d.entityId ? null : d.entityId)}
                className={`cursor-pointer ${expanded === d.entityId ? "!bg-surface" : ""}`}
              >
                <td>
                  <ClassIcon className={d.className} />
                  <span className="ml-2">{d.name}</span>
                </td>
                <td className="text-right">{formatAmount(d.totalDamageTaken)}</td>
                <td className="text-right">{formatDps(d.dtps)}</td>
              </tr>
              {expanded === d.entityId &&
                d.sources.map((s, i) => (
                  <tr key={`${d.entityId}-${i}`} className="bg-surface text-text-muted text-[0.8rem] [&>td]:border-b-border/50">
                    <td className="pl-8">
                      {s.source} — {formatAbilityName(s.abilityId)}
                    </td>
                    <td className="text-right">{formatAmount(s.totalDamage)}</td>
                    <td className="text-right">{s.hitCount} hits</td>
                  </tr>
                ))}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}
