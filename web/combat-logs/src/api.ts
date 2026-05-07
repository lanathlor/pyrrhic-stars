import type { InstanceLog, LogEntry, EncounterStats } from "./types";

const BASE = import.meta.env.VITE_API_URL || "";

function buildURL(path: string, params?: Record<string, string>): string {
  const qs = new URLSearchParams();
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      if (v) qs.set(k, v);
    });
  }
  const query = qs.toString();
  return `${BASE}${path}${query ? `?${query}` : ""}`;
}

export async function fetchInstances(params?: Record<string, string>): Promise<InstanceLog[]> {
  const res = await fetch(buildURL("/api/v1/logs/instances", params));
  if (!res.ok) throw new Error(`Failed to fetch instances: ${res.status}`);
  return res.json();
}

export async function fetchInstance(id: string): Promise<InstanceLog> {
  const res = await fetch(buildURL(`/api/v1/logs/instances/${id}`));
  if (!res.ok) throw new Error(`Failed to fetch instance: ${res.status}`);
  return res.json();
}

export async function fetchEvents(id: string, params?: Record<string, string>): Promise<LogEntry[]> {
  const res = await fetch(buildURL(`/api/v1/logs/instances/${id}/events`, params));
  if (!res.ok) throw new Error(`Failed to fetch events: ${res.status}`);
  return res.json();
}

export function exportURL(id: string): string {
  return `${BASE}/api/v1/logs/instances/${id}/export`;
}

export async function fetchEncounterStats(
  encounterId: string,
  params?: Record<string, string>
): Promise<EncounterStats> {
  const res = await fetch(
    buildURL(`/api/v1/logs/stats/encounter/${encounterId}`, params)
  );
  if (!res.ok) throw new Error(`Failed to fetch encounter stats: ${res.status}`);
  return res.json();
}
