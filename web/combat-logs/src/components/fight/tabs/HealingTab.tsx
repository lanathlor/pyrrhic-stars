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
      <div className="tab-content">
        <p style={{ color: "var(--text-muted)" }}>No healing events recorded.</p>
      </div>
    );
  }

  return (
    <div className="tab-content">
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th>Player</th>
            <th style={{ textAlign: "right" }}>Total</th>
            <th style={{ textAlign: "right" }}>HPS</th>
            <th style={{ textAlign: "right" }}>Crit%</th>
            <th style={{ textAlign: "right" }}>Hits</th>
          </tr>
        </thead>
        <tbody>
          {healingDone.map((h) => (
            <>
              <tr
                key={h.entityId}
                onClick={() => setExpanded(expanded === h.entityId ? null : h.entityId)}
                style={{ cursor: "pointer" }}
                className={expanded === h.entityId ? "row-expanded" : ""}
              >
                <td>
                  <ClassIcon className={h.className} />
                  <span style={{ marginLeft: "0.5rem" }}>{h.name}</span>
                </td>
                <td style={{ textAlign: "right" }}>{formatAmount(h.totalHealing)}</td>
                <td style={{ textAlign: "right" }}>{formatDps(h.hps)}</td>
                <td style={{ textAlign: "right" }}>{formatPercent(h.critRate)}</td>
                <td style={{ textAlign: "right" }}>{h.hitCount}</td>
              </tr>
              {expanded === h.entityId &&
                h.abilities.map((a) => (
                  <tr key={`${h.entityId}-${a.abilityId}`} className="row-ability">
                    <td style={{ paddingLeft: "2rem" }}>{formatAbilityName(a.abilityId)}</td>
                    <td style={{ textAlign: "right" }}>{formatAmount(a.totalDamage)}</td>
                    <td style={{ textAlign: "right" }}>—</td>
                    <td style={{ textAlign: "right" }}>{formatPercent(a.critRate)}</td>
                    <td style={{ textAlign: "right" }}>{a.hitCount}</td>
                  </tr>
                ))}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}
