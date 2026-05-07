interface Props {
  outcome: string;
}

const OUTCOME_CLASSES: Record<string, string> = {
  player_win: "text-success",
  boss_win: "text-danger",
};

export function OutcomeBadge({ outcome }: Props) {
  const cls = OUTCOME_CLASSES[outcome] ?? "text-warning";
  return (
    <span className={`font-semibold ${cls}`}>
      {outcome.replace(/_/g, " ")}
    </span>
  );
}
