import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import type { InstanceLog } from "../../types";
import { fetchInstances } from "../../api";
import { formatDuration } from "../../analysis/format";
import { OutcomeBadge } from "../shared/OutcomeBadge";
import { ClassIcon } from "../shared/ClassIcon";
import { FightListFilters } from "./FightListFilters";

export function FightListPage() {
  const [encounter, setEncounter] = useState("");
  const [outcome, setOutcome] = useState("");
  const [source, setSource] = useState("");

  const params: Record<string, string> = {};
  if (outcome) params.outcome = outcome;
  if (source) params.source = source;

  const { data: instances = [], isLoading, error } = useQuery({
    queryKey: ["instances", params],
    queryFn: () => fetchInstances(params),
  });

  const filtered = encounter
    ? instances.filter((i: InstanceLog) =>
        i.encounter_id.toLowerCase().includes(encounter.toLowerCase())
      )
    : instances;

  if (isLoading) return <p className="text-text-muted">Loading...</p>;
  if (error) return <p className="text-danger">{error.message}</p>;

  return (
    <div>
      <FightListFilters
        encounter={encounter}
        setEncounter={setEncounter}
        outcome={outcome}
        setOutcome={setOutcome}
        source={source}
        setSource={setSource}
      />
      {filtered.length === 0 ? (
        <p className="text-text-muted mt-4">No combat logs found.</p>
      ) : (
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th>Encounter</th>
              <th>Outcome</th>
              <th>Duration</th>
              <th>Players</th>
              <th>Source</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((inst: InstanceLog) => (
              <tr key={inst.instance_id}>
                <td className="font-medium">
                  <Link
                    to="/fight/$instanceId"
                    params={{ instanceId: inst.instance_id }}
                    className="hover:text-accent"
                  >
                    {inst.encounter_id}
                  </Link>
                </td>
                <td>
                  <OutcomeBadge outcome={inst.outcome} />
                </td>
                <td>{formatDuration(inst.duration_ms)}</td>
                <td>
                  <div className="flex gap-2 flex-wrap">
                    {(inst.participants ?? [])
                      .filter((p) => p.entity_id.startsWith("player"))
                      .map((p) => (
                        <span key={p.entity_id} className="inline-flex items-center gap-1.5 px-2.5 py-0.5 bg-surface border border-border rounded text-xs">
                          <ClassIcon className={p.class} showName={false} />
                          <span>{p.name}</span>
                        </span>
                      ))}
                  </div>
                </td>
                <td className="text-text-muted">{inst.source}</td>
                <td className="text-text-muted">
                  {new Date(inst.started_at).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
