import { useState, useMemo } from "react";
import { useFightContext } from "../../../hooks/FightContext";
import { EVENT_TYPE_NAMES } from "../../../types";
import { formatDuration, formatAbilityName } from "../../../analysis/format";
import { EVENT_TYPES } from "../../../constants";

const PAGE_SIZE = 100;

const btnCls = "px-4 py-1.5 border border-border rounded bg-surface text-text cursor-pointer text-sm hover:border-accent disabled:opacity-40 disabled:cursor-default";
const selectCls = "px-3 py-1.5 bg-surface border border-border rounded text-text text-sm";
const inputCls = "px-3 py-1.5 bg-surface border border-border rounded text-text text-sm min-w-[150px] placeholder:text-text-muted";

export function EventsTab() {
  const { instance, analysis } = useFightContext();
  const { filteredEvents } = analysis;

  const nameMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const p of instance.participants ?? []) {
      m.set(p.entity_id, p.name);
    }
    return m;
  }, [instance.participants]);

  const resolveName = (id: string) => {
    const name = nameMap.get(id);
    if (name) return name;
    if (id.startsWith("enemy_")) return formatAbilityName(instance.encounter_id);
    return id;
  };

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
    <div className="min-h-[200px]">
      <div className="flex flex-col gap-3 mb-4">
        <div className="flex flex-wrap gap-2">
          {Object.entries(EVENT_TYPE_NAMES).map(([key, name]) => {
            const k = Number(key);
            return (
              <label key={k} className="inline-flex items-center gap-1 text-[0.8rem] text-text-muted cursor-pointer">
                <input
                  type="checkbox"
                  checked={typeFilter.has(k)}
                  onChange={() => toggleType(k)}
                  className="accent-accent"
                />
                {name}
              </label>
            );
          })}
        </div>
        <div className="flex gap-2 flex-wrap">
          <select value={sourceFilter} onChange={(e) => { setSourceFilter(e.target.value); setPage(0); }} className={selectCls}>
            <option value="">All Sources</option>
            {sources.map((s) => <option key={s} value={s}>{resolveName(s)}</option>)}
          </select>
          <select value={targetFilter} onChange={(e) => { setTargetFilter(e.target.value); setPage(0); }} className={selectCls}>
            <option value="">All Targets</option>
            {targets.map((t) => <option key={t} value={t}>{resolveName(t)}</option>)}
          </select>
          <input
            type="text"
            placeholder="Filter ability..."
            value={abilityFilter}
            onChange={(e) => { setAbilityFilter(e.target.value); setPage(0); }}
            className={inputCls}
          />
        </div>
      </div>

      <div className="text-[0.8rem] text-text-muted mb-2">
        {filtered.length} events
        {totalPages > 1 && ` — page ${page + 1} of ${totalPages}`}
      </div>

      <div className="max-h-[600px] overflow-y-auto border border-border rounded-md">
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th>Time</th>
              <th>Type</th>
              <th>Source</th>
              <th>Target</th>
              <th>Ability</th>
              <th className="text-right">Amount</th>
              <th>Phase</th>
            </tr>
          </thead>
          <tbody>
            {pageEvents.map((e, i) => (
              <tr key={i} className={eventRowClass(e.event_type)}>
                <td>{formatDuration(e.timestamp_ms)}</td>
                <td>{EVENT_TYPE_NAMES[e.event_type] ?? e.event_type}</td>
                <td>{resolveName(e.source)}</td>
                <td>{e.target ? resolveName(e.target) : "—"}</td>
                <td>{formatAbilityName(e.ability_id)}</td>
                <td className="text-right">
                  {e.amount > 0 ? `${Math.round(e.amount).toLocaleString()}${e.is_crit ? "*" : ""}` : "—"}
                </td>
                <td>{e.phase || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4 mt-4 text-sm text-text-muted">
          <button className={btnCls} disabled={page === 0} onClick={() => setPage(page - 1)}>
            Prev
          </button>
          <span>
            {page + 1} / {totalPages}
          </span>
          <button className={btnCls} disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)}>
            Next
          </button>
        </div>
      )}
    </div>
  );
}

export function eventRowClass(eventType: number): string {
  switch (eventType) {
    case EVENT_TYPES.DAMAGE: return "[&>td]:text-danger";
    case EVENT_TYPES.HEAL: return "[&>td]:text-success";
    case EVENT_TYPES.DEATH: return "[&>td]:text-warning [&>td]:font-semibold";
    default: return "";
  }
}
