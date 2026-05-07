import { FightTabs } from "./FightTabs";
import { PhaseSelector } from "./PhaseSelector";
import { useFightContext } from "../../hooks/FightContext";

interface Props {
  instanceId: string;
}

export function FightTabsBar({ instanceId }: Props) {
  const { analysis, selectedPhase, setSelectedPhase } = useFightContext();

  return (
    <div className="flex items-center gap-2 mb-6 border-b border-border">
      <FightTabs instanceId={instanceId} />
      {analysis.phases.length > 0 && (
        <PhaseSelector
          phases={analysis.phases}
          selected={selectedPhase}
          onSelect={setSelectedPhase}
        />
      )}
    </div>
  );
}
