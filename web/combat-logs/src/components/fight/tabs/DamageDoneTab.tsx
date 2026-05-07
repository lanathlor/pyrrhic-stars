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
    fill: CLASS_COLORS[d.className] ?? "var(--color-accent)",
  }));

  return (
    <div className="min-h-[200px]">
      <div className="grid grid-cols-[1fr_280px] gap-6 items-start max-[900px]:grid-cols-1">
        <div className="min-w-0">
          <table className="w-full border-collapse">
            <thead>
              <tr>
                <th>Player</th>
                <th className="text-right">Total</th>
                <th className="text-right">DPS</th>
                <th className="text-right">Crit%</th>
                <th className="text-right">Hits</th>
                <th className="text-right">%</th>
              </tr>
            </thead>
            <tbody>
              {damageDone.map((d) => (
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
                    <td className="text-right">{formatAmount(d.totalDamage)}</td>
                    <td className="text-right">{formatDps(d.dps)}</td>
                    <td className="text-right">{formatPercent(d.critRate)}</td>
                    <td className="text-right">{d.hitCount}</td>
                    <td className="text-right">
                      {totalDamage > 0 ? formatPercent(d.totalDamage / totalDamage) : "—"}
                    </td>
                  </tr>
                  {expanded === d.entityId &&
                    d.abilities.map((a) => (
                      <tr key={`${d.entityId}-${a.abilityId}`} className="bg-surface text-text-muted text-[0.8rem] [&>td]:border-b-border/50">
                        <td className="pl-8">{formatAbilityName(a.abilityId)}</td>
                        <td className="text-right">{formatAmount(a.totalDamage)}</td>
                        <td className="text-right">—</td>
                        <td className="text-right">{formatPercent(a.critRate)}</td>
                        <td className="text-right">{a.hitCount}</td>
                        <td className="text-right">
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
          <div className="sticky top-4">
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
                  stroke="var(--color-bg)"
                >
                  {pieData.map((entry, i) => (
                    <Cell key={i} fill={entry.fill} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    backgroundColor: "var(--color-surface)",
                    border: "1px solid var(--color-border)",
                    borderRadius: 4,
                    color: "var(--color-text)",
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
