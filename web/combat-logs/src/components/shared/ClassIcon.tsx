import { CLASS_COLORS, CLASS_DISPLAY_NAMES } from "../../constants";

interface Props {
  className: string;
  showName?: boolean;
}

export function ClassIcon({ className, showName = true }: Props) {
  const color = CLASS_COLORS[className] ?? "var(--color-text-muted)";
  return (
    <span className="inline-flex items-center gap-1.5">
      <span
        className="size-2 rounded-full inline-block shrink-0"
        style={{ backgroundColor: color }}
      />
      {showName && (
        <span style={{ color }}>{CLASS_DISPLAY_NAMES[className] ?? className}</span>
      )}
    </span>
  );
}
