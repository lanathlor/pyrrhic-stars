import type { PhaseMarker } from "../../types";

interface Props {
  phases: PhaseMarker[];
  selected: string | null;
  onSelect: (phase: string | null) => void;
}

export function PhaseSelector({ phases, selected, onSelect }: Props) {
  return (
    <select
      className="phase-selector"
      value={selected ?? ""}
      onChange={(e) => onSelect(e.target.value || null)}
    >
      <option value="">All Phases</option>
      {phases.map((p) => (
        <option key={p.phase} value={p.phase}>
          {p.phase}
        </option>
      ))}
    </select>
  );
}
