import { CLASS_COLORS } from "../../constants";

interface Props {
  name: string;
  className: string;
  value: string;
  secondary?: string;
  percent: number; // 0-1 relative to max
}

export function DamageBar({ name, className, value, secondary, percent }: Props) {
  const color = CLASS_COLORS[className] ?? "var(--accent)";
  return (
    <div className="damage-bar">
      <div
        className="damage-bar-fill"
        style={{ width: `${Math.max(percent * 100, 2)}%`, backgroundColor: color }}
      />
      <div className="damage-bar-content">
        <span className="damage-bar-name">{name}</span>
        <span className="damage-bar-values">
          <span className="damage-bar-value">{value}</span>
          {secondary && <span className="damage-bar-secondary">{secondary}</span>}
        </span>
      </div>
    </div>
  );
}
