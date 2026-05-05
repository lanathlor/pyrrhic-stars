import { useState, useMemo } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { EVENT_TYPE_NAMES } from "../../../types";
import { formatDuration, formatAbilityName } from "../../../analysis/format";
import { EVENT_TYPES } from "../../../constants";

const PAGE_SIZE = 100;

export function EventsTab() {
  const { analysis } = useFightContext();
  const { filteredEvents } = analysis;

  const [typeFilter, setTypeFilter] = useState<Set<number>>(new Set());
  const [sourceFilter, setSourceFilter] = useState("");
  const [targetFilter, setTargetFilter] = useState("");
  const [abilityFilter, setAbilityFilter] = useState("");
  const [page, setPage] = useState(0);

  const sources = useMemo(
    () => Array.from(new Set(filteredEvents.map((e) => e.source).filter(Boolean))).sort(),
    [filteredEvents]
  );
  const targets = useMemo(
    () => Array.from(new Set(filteredEvents.map((e) => e.target).filter(Boolean))).sort(),
    [filteredEvents]
  );

  const filtered = useMemo(() => {
    return filteredEvents.filter((e) => {
      if (typeFilter.size > 0 && !typeFilter.has(e.event_type)) return false;
      if (sourceFilter && e.source !== sourceFilter) return false;
      if (targetFilter && e.target !== targetFilter) return false;
      if (abilityFilter && !e.ability_id.toLowerCase().includes(abilityFilter.toLowerCase())) return false;
      return true;
    });
  }, [filteredEvents, typeFilter, sourceFilter, targetFilter, abilityFilter]);

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const pageEvents = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const toggleType = (t: number) => {
    setTypeFilter((prev) => {
      const next = new Set(prev);
      if (next.has(t)) next.delete(t);
      else next.add(t);
      return next;
    });
    setPage(0);
  };

  return (
    <div className="tab-content">
      <div className="filter-bar">
        <div className="event-type-filters">
          {Object.entries(EVENT_TYPE_NAMES).map(([key, name]) => {
            const k = Number(key);
            return (
              <label key={k} className="event-type-check">
                <input
                  type="checkbox"
                  checked={typeFilter.has(k)}
                  onChange={() => toggleType(k)}
                />
                {name}
              </label>
            );
          })}
        </div>
        <div className="filter-row">
          <select value={sourceFilter} onChange={(e) => { setSourceFilter(e.target.value); setPage(0); }} className="filter-select">
            <option value="">All Sources</option>
            {sources.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
          <select value={targetFilter} onChange={(e) => { setTargetFilter(e.target.value); setPage(0); }} className="filter-select">
            <option value="">All Targets</option>
            {targets.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
          <input
            type="text"
            placeholder="Filter ability..."
            value={abilityFilter}
            onChange={(e) => { setAbilityFilter(e.target.value); setPage(0); }}
            className="filter-input"
          />
        </div>
      </div>

      <div className="events-meta">
        {filtered.length} events
        {totalPages > 1 && ` — page ${page + 1} of ${totalPages}`}
      </div>

      <div className="events-table-wrapper">
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr>
              <th>Time</th>
              <th>Type</th>
              <th>Source</th>
              <th>Target</th>
              <th>Ability</th>
              <th style={{ textAlign: "right" }}>Amount</th>
              <th>Phase</th>
            </tr>
          </thead>
          <tbody>
            {pageEvents.map((e, i) => (
              <tr key={i} className={eventRowClass(e.event_type)}>
                <td>{formatDuration(e.timestamp_ms)}</td>
                <td>{EVENT_TYPE_NAMES[e.event_type] ?? e.event_type}</td>
                <td>{e.source}</td>
                <td>{e.target || "—"}</td>
                <td>{formatAbilityName(e.ability_id)}</td>
                <td style={{ textAlign: "right" }}>
                  {e.amount > 0 ? `${Math.round(e.amount).toLocaleString()}${e.is_crit ? "*" : ""}` : "—"}
                </td>
                <td>{e.phase || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="pagination">
          <button className="btn" disabled={page === 0} onClick={() => setPage(page - 1)}>
            Prev
          </button>
          <span>
            {page + 1} / {totalPages}
          </span>
          <button className="btn" disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)}>
            Next
          </button>
        </div>
      )}
    </div>
  );
}

function eventRowClass(eventType: number): string {
  switch (eventType) {
    case EVENT_TYPES.DAMAGE: return "row-damage";
    case EVENT_TYPES.HEAL: return "row-heal";
    case EVENT_TYPES.DEATH: return "row-death";
    default: return "";
  }
}
