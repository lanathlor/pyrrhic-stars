import { useFightContext } from "../../../hooks/FightContext";
import { formatDuration, formatAmount, formatAbilityName } from "../../../analysis/format";
import { ClassIcon } from "../../shared/ClassIcon";
import { EVENT_TYPE_NAMES } from "../../../types";

export function DeathsTab() {
  const { analysis } = useFightContext();
  const { deaths } = analysis;

  if (deaths.length === 0) {
    return (
      <div className="tab-content">
        <p style={{ color: "var(--text-muted)" }}>No player deaths recorded.</p>
      </div>
    );
  }

  return (
    <div className="tab-content">
      <div className="death-timeline">
        {deaths.map((d, i) => (
          <div key={i} className="death-card">
            <div className="death-card-header">
              <span className="death-time">{formatDuration(d.timestampMs)}</span>
              <ClassIcon className={d.victimClass} />
              <span style={{ fontWeight: 600, color: "var(--warning)" }}>{d.victimName}</span>
              {d.killingBlow && (
                <span style={{ color: "var(--text-muted)", marginLeft: "auto" }}>
                  Killing blow: <strong>{formatAbilityName(d.killingBlow.ability_id)}</strong>
                  {" "}from {d.killingBlow.source}
                  {" "}({formatAmount(d.killingBlow.amount)})
                </span>
              )}
            </div>
            <div className="death-card-events">
              {d.leadup.map((e, j) => (
                <div
                  key={j}
                  className={`death-event ${
                    e.event_type === 1
                      ? "death-event-damage"
                      : e.event_type === 2
                        ? "death-event-heal"
                        : ""
                  }`}
                >
                  <span className="death-event-time">{formatDuration(e.timestamp_ms)}</span>
                  <span className="death-event-type">{EVENT_TYPE_NAMES[e.event_type] ?? e.event_type}</span>
                  <span>{e.source} → {e.target || "—"}</span>
                  <span>{formatAbilityName(e.ability_id)}</span>
                  {e.amount > 0 && (
                    <span style={{ fontWeight: 500 }}>
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
