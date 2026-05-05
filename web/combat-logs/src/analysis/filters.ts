import type { LogEntry } from "../types";

export function filterByPhase(events: LogEntry[], phase: string | null): LogEntry[] {
  if (!phase) return events;
  return events.filter((e) => e.phase === phase);
}

export function filterByPlayer(events: LogEntry[], entityId: string): LogEntry[] {
  return events.filter((e) => e.source === entityId || e.target === entityId);
}

export function filterByType(events: LogEntry[], types: number[]): LogEntry[] {
  const set = new Set(types);
  return events.filter((e) => set.has(e.event_type));
}
