interface Props {
  outcome: string;
}

export function OutcomeBadge({ outcome }: Props) {
  const color =
    outcome === "player_win"
      ? "var(--success)"
      : outcome === "boss_win"
        ? "var(--danger)"
        : "var(--warning)";
  return (
    <span style={{ color, fontWeight: 600 }}>
      {outcome.replace(/_/g, " ")}
    </span>
  );
}
