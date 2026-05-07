import { useState, useMemo } from "react";
import { Link } from "@tanstack/react-router";
import type { InstanceLog } from "../../types";
import { formatDuration } from "../../analysis/format";
import { OutcomeBadge } from "../shared/OutcomeBadge";
import { ClassIcon } from "../shared/ClassIcon";

interface Props {
  instances: InstanceLog[];
}

type SortKey = "date" | "duration" | "outcome";
type SortDir = "asc" | "desc";

const PAGE_SIZE = 50;

export function RunsTable({ instances }: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("date");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [outcomeFilter, setOutcomeFilter] = useState("");
  const [profileFilter, setProfileFilter] = useState("");
  const [page, setPage] = useState(0);

  const profiles = useMemo(() => {
    const set = new Set<string>();
    for (const inst of instances) {
      for (const p of inst.participants ?? []) {
        if (p.bot_profile) set.add(p.bot_profile);
      }
    }
    return Array.from(set).sort();
  }, [instances]);

  const filtered = useMemo(() => {
    let list = instances;
    if (outcomeFilter) {
      list = list.filter((i) => i.outcome === outcomeFilter);
    }
    if (profileFilter) {
      list = list.filter((i) =>
        (i.participants ?? []).some((p) => p.bot_profile === profileFilter)
      );
    }
    return list;
  }, [instances, outcomeFilter, profileFilter]);

  const sorted = useMemo(() => {
    const copy = [...filtered];
    copy.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case "date":
          cmp = a.started_at.localeCompare(b.started_at);
          break;
        case "duration":
          cmp = a.duration_ms - b.duration_ms;
          break;
        case "outcome":
          cmp = a.outcome.localeCompare(b.outcome);
          break;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });
    return copy;
  }, [filtered, sortKey, sortDir]);

  const totalPages = Math.ceil(sorted.length / PAGE_SIZE);
  const pageItems = sorted.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
    setPage(0);
  };

  const sortIndicator = (key: SortKey) =>
    sortKey === key ? (sortDir === "asc" ? " ↑" : " ↓") : "";

  return (
    <div>
      <div className="flex gap-2 mb-3">
        <select
          value={outcomeFilter}
          onChange={(e) => { setOutcomeFilter(e.target.value); setPage(0); }}
          className="px-3 py-1.5 bg-surface border border-border rounded text-text text-sm"
        >
          <option value="">All Outcomes</option>
          <option value="player_win">Win</option>
          <option value="boss_win">Loss</option>
          <option value="timeout">Timeout</option>
        </select>
        {profiles.length > 1 && (
          <select
            value={profileFilter}
            onChange={(e) => { setProfileFilter(e.target.value); setPage(0); }}
            className="px-3 py-1.5 bg-surface border border-border rounded text-text text-sm"
          >
            <option value="">All Profiles</option>
            {profiles.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
        )}
        <span className="text-text-muted text-sm self-center ml-2">
          {filtered.length} runs
        </span>
      </div>

      <div className="max-h-[600px] overflow-y-auto border border-border rounded-md">
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th className="cursor-pointer select-none" onClick={() => toggleSort("date")}>
                Date{sortIndicator("date")}
              </th>
              <th className="cursor-pointer select-none" onClick={() => toggleSort("outcome")}>
                Outcome{sortIndicator("outcome")}
              </th>
              <th className="cursor-pointer select-none text-right" onClick={() => toggleSort("duration")}>
                Duration{sortIndicator("duration")}
              </th>
              <th>Players</th>
              <th>Profile</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {pageItems.map((inst) => {
              const players = (inst.participants ?? []).filter((p) => p.entity_id.startsWith("player"));
              const profile = players[0]?.bot_profile ?? "—";
              return (
                <tr key={inst.instance_id}>
                  <td className="text-text-muted tabular-nums">
                    {new Date(inst.started_at).toLocaleString()}
                  </td>
                  <td><OutcomeBadge outcome={inst.outcome} /></td>
                  <td className="text-right tabular-nums">{formatDuration(inst.duration_ms)}</td>
                  <td>
                    <div className="flex gap-1">
                      {players.map((p) => (
                        <ClassIcon key={p.entity_id} className={p.class} showName={false} />
                      ))}
                    </div>
                  </td>
                  <td>
                    <span className="px-1.5 py-0.5 bg-bg rounded text-xs text-text-muted">
                      {profile}
                    </span>
                  </td>
                  <td>
                    <Link
                      to="/fight/$instanceId"
                      params={{ instanceId: inst.instance_id }}
                      className="text-accent text-sm hover:underline"
                    >
                      View →
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4 mt-4 text-sm text-text-muted">
          <button
            className="px-4 py-1.5 border border-border rounded bg-surface text-text cursor-pointer text-sm hover:border-accent disabled:opacity-40 disabled:cursor-default"
            disabled={page === 0}
            onClick={() => setPage(page - 1)}
          >
            Prev
          </button>
          <span>{page + 1} / {totalPages}</span>
          <button
            className="px-4 py-1.5 border border-border rounded bg-surface text-text cursor-pointer text-sm hover:border-accent disabled:opacity-40 disabled:cursor-default"
            disabled={page >= totalPages - 1}
            onClick={() => setPage(page + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
