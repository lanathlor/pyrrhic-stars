import type { InstanceLog } from "../../types";
import { formatDuration } from "../../analysis/format";
import { OutcomeBadge } from "../shared/OutcomeBadge";
import { ClassIcon } from "../shared/ClassIcon";
import { useFightContext } from "../../hooks/FightContext";

interface Props {
  instance: InstanceLog;
}

export function FightHeader({ instance }: Props) {
  const { analysis } = useFightContext();
  const players = (instance.participants ?? []).filter((p) =>
    p.entity_id.startsWith("player")
  );

  return (
    <div className="mb-6">
      <div className="flex items-center gap-4 mb-3 flex-wrap">
        <h2>{instance.encounter_id}</h2>
        <OutcomeBadge outcome={instance.outcome} />
        <span className="text-text-muted text-sm">
          {formatDuration(analysis.effectiveDurationMs)}
        </span>
        <span className="text-text-muted text-sm">
          {new Date(instance.started_at).toLocaleString()}
        </span>
      </div>
      {players.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {players.map((p) => (
            <span key={p.entity_id} className="inline-flex items-center gap-1.5 px-2.5 py-0.5 bg-surface border border-border rounded text-xs">
              <ClassIcon className={p.class} />
              <span>{p.name}</span>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
