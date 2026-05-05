import { CLASS_COLORS, CLASS_DISPLAY_NAMES } from "../../constants";

interface Props {
  className: string;
  showName?: boolean;
}

export function ClassIcon({ className, showName = true }: Props) {
  const color = CLASS_COLORS[className] ?? "var(--text-muted)";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: "0.35rem" }}>
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: "50%",
          backgroundColor: color,
          display: "inline-block",
          flexShrink: 0,
        }}
      />
      {showName && (
        <span style={{ color }}>{CLASS_DISPLAY_NAMES[className] ?? className}</span>
      )}
    </span>
  );
}
