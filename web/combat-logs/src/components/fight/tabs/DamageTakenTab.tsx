import { useState } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { formatAmount, formatDps, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";

export function DamageTakenTab() {
  const { analysis } = useFightContext();
  const { damageTaken } = analysis;
  const [expanded, setExpanded] = useState<string | null>(null);

  return (
    <div className="tab-content">
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th>Player</th>
            <th style={{ textAlign: "right" }}>Total Taken</th>
            <th style={{ textAlign: "right" }}>DTPS</th>
          </tr>
        </thead>
        <tbody>
          {damageTaken.map((d) => (
            <>
              <tr
                key={d.entityId}
                onClick={() => setExpanded(expanded === d.entityId ? null : d.entityId)}
                style={{ cursor: "pointer" }}
                className={expanded === d.entityId ? "row-expanded" : ""}
              >
                <td>
                  <ClassIcon className={d.className} />
                  <span style={{ marginLeft: "0.5rem" }}>{d.name}</span>
                </td>
                <td style={{ textAlign: "right" }}>{formatAmount(d.totalDamageTaken)}</td>
                <td style={{ textAlign: "right" }}>{formatDps(d.dtps)}</td>
              </tr>
              {expanded === d.entityId &&
                d.sources.map((s, i) => (
                  <tr key={`${d.entityId}-${i}`} className="row-ability">
                    <td style={{ paddingLeft: "2rem" }}>
                      {s.source} — {formatAbilityName(s.abilityId)}
                    </td>
                    <td style={{ textAlign: "right" }}>{formatAmount(s.totalDamage)}</td>
                    <td style={{ textAlign: "right" }}>{s.hitCount} hits</td>
                  </tr>
                ))}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}
