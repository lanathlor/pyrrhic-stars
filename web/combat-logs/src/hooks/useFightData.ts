import { useEffect, useState } from "react";
import type { InstanceLog, LogEntry } from "../types";
import { fetchInstance, fetchEvents } from "../api";

export function useFightData(instanceId: string) {
  const [instance, setInstance] = useState<InstanceLog | null>(null);
  const [events, setEvents] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true);
    setError("");
    Promise.all([fetchInstance(instanceId), fetchEvents(instanceId)])
      .then(([inst, evts]) => {
        setInstance(inst);
        setEvents(evts);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [instanceId]);

  return { instance, events, loading, error };
}
