import { CLASS_COLORS } from "../../constants";

interface Props {
  name: string;
  className: string;
  value: string;
  secondary?: string;
  percent: number; // 0-1 relative to max
}

export function DamageBar({ name, className, value, secondary, percent }: Props) {
  const color = CLASS_COLORS[className] ?? "var(--color-accent)";
  return (
    <div className="relative h-8 bg-surface border border-border rounded overflow-hidden">
      <div
        className="absolute top-0 left-0 h-full opacity-25 transition-[width] duration-300 ease-out"
        style={{ width: `${Math.max(percent * 100, 2)}%`, backgroundColor: color }}
      />
      <div className="relative flex items-center justify-between h-full px-3 text-sm z-1">
        <span className="font-medium">{name}</span>
        <span className="flex gap-4 items-center">
          <span className="font-semibold">{value}</span>
          {secondary && <span className="text-text-muted text-[0.8rem]">{secondary}</span>}
        </span>
      </div>
    </div>
  );
}
