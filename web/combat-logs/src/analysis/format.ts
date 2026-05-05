export function formatDuration(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
  }
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export function formatTimestamp(ms: number): string {
  const s = ms / 1000;
  return s.toFixed(1) + "s";
}

export function formatAmount(n: number): string {
  return Math.round(n).toLocaleString();
}

export function formatDps(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
  return Math.round(n).toString();
}

export function formatPercent(n: number): string {
  return (n * 100).toFixed(1) + "%";
}

export function formatAbilityName(id: string): string {
  if (!id) return "Auto Attack";
  return id
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
