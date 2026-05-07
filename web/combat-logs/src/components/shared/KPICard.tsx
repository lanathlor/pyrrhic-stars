interface Props {
  label: string;
  value: string;
  subtitle?: string;
}

export function KPICard({ label, value, subtitle }: Props) {
  return (
    <div className="flex flex-col px-4 py-3 bg-surface border border-border rounded-md">
      <span className="text-[0.7rem] uppercase tracking-wide text-text-muted mb-1">{label}</span>
      <span className="text-xl font-semibold">{value}</span>
      {subtitle && <span className="text-xs text-text-muted mt-0.5">{subtitle}</span>}
    </div>
  );
}
