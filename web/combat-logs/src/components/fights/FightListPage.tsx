import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { InstanceLog } from "../../types";
import { fetchInstances } from "../../api";
import { formatDuration } from "../../analysis/format";
import { OutcomeBadge } from "../shared/OutcomeBadge";
import { ClassIcon } from "../shared/ClassIcon";
import { FightListFilters } from "./FightListFilters";

export function FightListPage() {
  const navigate = useNavigate();
  const [instances, setInstances] = useState<InstanceLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [encounter, setEncounter] = useState("");
  const [outcome, setOutcome] = useState("");
  const [source, setSource] = useState("");

  useEffect(() => {
    setLoading(true);
    const params: Record<string, string> = {};
    if (outcome) params.outcome = outcome;
    if (source) params.source = source;

    fetchInstances(params)
      .then(setInstances)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [outcome, source]);

  const filtered = encounter
    ? instances.filter((i) =>
        i.encounter_id.toLowerCase().includes(encounter.toLowerCase())
      )
    : instances;

  if (loading) return <p style={{ color: "var(--text-muted)" }}>Loading...</p>;
  if (error) return <p style={{ color: "var(--danger)" }}>{error}</p>;

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
        <p style={{ color: "var(--text-muted)", marginTop: "1rem" }}>No combat logs found.</p>
      ) : (
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
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
            {filtered.map((inst) => (
              <tr
                key={inst.instance_id}
                onClick={() => navigate(`/fight/${inst.instance_id}`)}
                style={{ cursor: "pointer" }}
              >
                <td style={{ fontWeight: 500 }}>{inst.encounter_id}</td>
                <td>
                  <OutcomeBadge outcome={inst.outcome} />
                </td>
                <td>{formatDuration(inst.duration_ms)}</td>
                <td>
                  <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
                    {(inst.participants ?? [])
                      .filter((p) => p.entity_id.startsWith("player"))
                      .map((p) => (
                        <span key={p.entity_id} className="participant-chip">
                          <ClassIcon className={p.class} showName={false} />
                          <span>{p.name}</span>
                        </span>
                      ))}
                  </div>
                </td>
                <td style={{ color: "var(--text-muted)" }}>{inst.source}</td>
                <td style={{ color: "var(--text-muted)" }}>
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
