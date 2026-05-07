import { useFightContext } from "../../../hooks/FightContext";
import { formatDuration, formatAmount, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";
import { EVENT_TYPE_NAMES } from "../../../types";

export function DeathsTab() {
  const { instance, analysis } = useFightContext();
  const { deaths } = analysis;

  const nameMap = new Map((instance.participants ?? []).map((p) => [p.entity_id, p.name]));
  const resolveName = (id: string) => {
    const name = nameMap.get(id);
    if (name) return name;
    if (id.startsWith("enemy_")) return formatAbilityName(instance.encounter_id);
    return id;
  };

  if (deaths.length === 0) {
    return (
      <div className="min-h-[200px]">
        <p className="text-text-muted">No player deaths recorded.</p>
      </div>
    );
  }

  return (
    <div className="min-h-[200px]">
      <div className="flex flex-col gap-4">
        {deaths.map((d, i) => (
          <div key={i} className="bg-surface border border-border rounded-md overflow-hidden">
            <div className="flex items-center gap-3 px-4 py-3 border-b border-border text-sm">
              <span className="text-text-muted tabular-nums min-w-12">{formatDuration(d.timestampMs)}</span>
              <ClassIcon className={d.victimClass} />
              <span className="font-semibold text-warning">{d.victimName}</span>
              {d.killingBlow && (
                <span className="text-text-muted ml-auto">
                  Killing blow: <strong>{formatAbilityName(d.killingBlow.ability_id)}</strong>
                  {" "}from {resolveName(d.killingBlow.source)}
                  {" "}({formatAmount(d.killingBlow.amount)})
                </span>
              )}
            </div>
            <div className="p-2 text-[0.8rem]">
              {d.leadup.map((e, j) => (
                <div
                  key={j}
                  className={`flex gap-3 px-2 py-0.5 rounded-sm hover:bg-surface-hover ${
                    e.event_type === 1
                      ? "text-danger"
                      : e.event_type === 2
                        ? "text-success"
                        : ""
                  }`}
                >
                  <span className="text-text-muted min-w-12 tabular-nums">{formatDuration(e.timestamp_ms)}</span>
                  <span className="min-w-20 text-text-muted">{EVENT_TYPE_NAMES[e.event_type] ?? e.event_type}</span>
                  <span>{resolveName(e.source)} → {e.target ? resolveName(e.target) : "—"}</span>
                  <span>{formatAbilityName(e.ability_id)}</span>
                  {e.amount > 0 && (
                    <span className="font-medium">
                      {formatAmount(e.amount)}{e.is_crit ? "*" : ""}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
