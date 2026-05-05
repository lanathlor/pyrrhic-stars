import { useFightContext } from "../../../hooks/FightContext";
import { formatDps } from "../../../analysis/format";
import { CLASS_COLORS } from "../../../constants";
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
} from "recharts";

export function TimelineTab() {
  const { instance, analysis } = useFightContext();
  const { bossHP, dpsTimeline, phases } = analysis;
  const participants = instance.participants ?? [];
  const playerParticipants = participants.filter((p) => p.entity_id.startsWith("player"));

  const formatTime = (ms: number) => {
    const s = Math.floor(ms / 1000);
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${m}:${sec.toString().padStart(2, "0")}`;
  };

  const tooltipStyle = {
    backgroundColor: "var(--surface)",
    border: "1px solid var(--border)",
    borderRadius: 4,
    color: "var(--text)",
  };

  return (
    <div className="tab-content">
      {bossHP.length > 0 && (
        <section>
          <h3>Boss Health</h3>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={bossHP}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis
                dataKey="timestampMs"
                tickFormatter={formatTime}
                stroke="var(--text-muted)"
                fontSize={12}
              />
              <YAxis
                domain={[0, 100]}
                tickFormatter={(v) => `${v}%`}
                stroke="var(--text-muted)"
                fontSize={12}
              />
              <Tooltip
                contentStyle={tooltipStyle}
                labelFormatter={(label) => formatTime(Number(label))}
                formatter={(value) => [`${Number(value).toFixed(1)}%`, "Boss HP"]}
              />
              {phases.map((p) => (
                <ReferenceLine
                  key={p.phase}
                  x={p.startMs}
                  stroke="var(--text-muted)"
                  strokeDasharray="3 3"
                  label={{ value: p.phase, position: "top", fill: "var(--text-muted)", fontSize: 11 }}
                />
              ))}
              <defs>
                <linearGradient id="bossHPGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--danger)" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="var(--danger)" stopOpacity={0.05} />
                </linearGradient>
              </defs>
              <Area
                type="monotone"
                dataKey="value"
                stroke="var(--danger)"
                fill="url(#bossHPGradient)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        </section>
      )}

      {dpsTimeline.length > 0 && (
        <section style={{ marginTop: "1.5rem" }}>
          <h3>DPS Over Time</h3>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={dpsTimeline}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis
                dataKey="timestampMs"
                tickFormatter={formatTime}
                stroke="var(--text-muted)"
                fontSize={12}
              />
              <YAxis
                tickFormatter={(v) => formatDps(v)}
                stroke="var(--text-muted)"
                fontSize={12}
              />
              <Tooltip
                contentStyle={tooltipStyle}
                labelFormatter={(label) => formatTime(Number(label))}
                formatter={(value, name) => {
                  const p = participants.find((pp) => pp.entity_id === name);
                  return [formatDps(Number(value)), p?.name ?? String(name)];
                }}
              />
              {phases.map((p) => (
                <ReferenceLine
                  key={p.phase}
                  x={p.startMs}
                  stroke="var(--text-muted)"
                  strokeDasharray="3 3"
                  label={{ value: p.phase, position: "top", fill: "var(--text-muted)", fontSize: 11 }}
                />
              ))}
              {playerParticipants.map((p) => (
                <Line
                  key={p.entity_id}
                  type="monotone"
                  dataKey={p.entity_id}
                  stroke={CLASS_COLORS[p.class] ?? "var(--accent)"}
                  strokeWidth={1.5}
                  dot={false}
                  name={p.entity_id}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </section>
      )}
    </div>
  );
}
