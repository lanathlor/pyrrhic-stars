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
    <div className="fight-header">
      <div className="fight-header-top">
        <h2>{instance.encounter_id}</h2>
        <OutcomeBadge outcome={instance.outcome} />
        <span className="fight-header-meta">
          {formatDuration(analysis.effectiveDurationMs)}
        </span>
        <span className="fight-header-meta">
          {new Date(instance.started_at).toLocaleString()}
        </span>
      </div>
      {players.length > 0 && (
        <div className="fight-header-players">
          {players.map((p) => (
            <span key={p.entity_id} className="participant-chip">
              <ClassIcon className={p.class} />
              <span>{p.name}</span>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
