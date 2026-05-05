import { useState } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { formatAmount, formatDps, formatPercent, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";
import { CLASS_COLORS } from "../../../constants";

export function DamageDoneTab() {
  const { analysis } = useFightContext();
  const { damageDone } = analysis;
  const [expanded, setExpanded] = useState<string | null>(null);

  const totalDamage = damageDone.reduce((s, d) => s + d.totalDamage, 0);
  const pieData = damageDone.map((d) => ({
    name: d.name,
    value: d.totalDamage,
    fill: CLASS_COLORS[d.className] ?? "var(--accent)",
  }));

  return (
    <div className="tab-content">
      <div className="tab-split">
        <div className="tab-main">
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr>
                <th>Player</th>
                <th style={{ textAlign: "right" }}>Total</th>
                <th style={{ textAlign: "right" }}>DPS</th>
                <th style={{ textAlign: "right" }}>Crit%</th>
                <th style={{ textAlign: "right" }}>Hits</th>
                <th style={{ textAlign: "right" }}>%</th>
              </tr>
            </thead>
            <tbody>
              {damageDone.map((d) => (
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
                    <td style={{ textAlign: "right" }}>{formatAmount(d.totalDamage)}</td>
                    <td style={{ textAlign: "right" }}>{formatDps(d.dps)}</td>
                    <td style={{ textAlign: "right" }}>{formatPercent(d.critRate)}</td>
                    <td style={{ textAlign: "right" }}>{d.hitCount}</td>
                    <td style={{ textAlign: "right" }}>
                      {totalDamage > 0 ? formatPercent(d.totalDamage / totalDamage) : "—"}
                    </td>
                  </tr>
                  {expanded === d.entityId &&
                    d.abilities.map((a) => (
                      <tr key={`${d.entityId}-${a.abilityId}`} className="row-ability">
                        <td style={{ paddingLeft: "2rem" }}>{formatAbilityName(a.abilityId)}</td>
                        <td style={{ textAlign: "right" }}>{formatAmount(a.totalDamage)}</td>
                        <td style={{ textAlign: "right" }}>—</td>
                        <td style={{ textAlign: "right" }}>{formatPercent(a.critRate)}</td>
                        <td style={{ textAlign: "right" }}>{a.hitCount}</td>
                        <td style={{ textAlign: "right" }}>
                          {d.totalDamage > 0 ? formatPercent(a.totalDamage / d.totalDamage) : "—"}
                        </td>
                      </tr>
                    ))}
                </>
              ))}
            </tbody>
          </table>
        </div>
        {pieData.length > 0 && (
          <div className="tab-side">
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={pieData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  strokeWidth={1}
                  stroke="var(--bg)"
                >
                  {pieData.map((entry, i) => (
                    <Cell key={i} fill={entry.fill} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    backgroundColor: "var(--surface)",
                    border: "1px solid var(--border)",
                    borderRadius: 4,
                    color: "var(--text)",
                  }}
                  formatter={(value) => formatAmount(Number(value))}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
}
