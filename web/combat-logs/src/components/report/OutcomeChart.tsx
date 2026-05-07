import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from "recharts";
import { formatPercent } from "../../analysis/format";

interface Props {
  wins: number;
  losses: number;
  timeouts: number;
}

const COLORS = {
  wins: "var(--color-success)",
  losses: "var(--color-danger)",
  timeouts: "var(--color-warning)",
};

export function OutcomeChart({ wins, losses, timeouts }: Props) {
  const total = wins + losses + timeouts;
  const data = [
    { name: "Wins", value: wins, fill: COLORS.wins },
    { name: "Losses", value: losses, fill: COLORS.losses },
    ...(timeouts > 0 ? [{ name: "Timeouts", value: timeouts, fill: COLORS.timeouts }] : []),
  ];

  return (
    <div className="bg-surface border border-border rounded-lg p-4">
      <h4 className="text-sm font-medium text-text-muted mb-2">Outcome Distribution</h4>
      <ResponsiveContainer width="100%" height={220}>
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            strokeWidth={2}
            stroke="var(--color-bg)"
          >
            {data.map((entry, i) => (
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
            formatter={(value) => [`${value} (${formatPercent(Number(value) / total)})`, ""]}
          />
          <Legend
            formatter={(value) => <span className="text-sm text-text">{value}</span>}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
