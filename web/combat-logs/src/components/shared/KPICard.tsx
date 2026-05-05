interface Props {
  label: string;
  value: string;
  subtitle?: string;
}

export function KPICard({ label, value, subtitle }: Props) {
  return (
    <div className="kpi-card">
      <span className="kpi-label">{label}</span>
      <span className="kpi-value">{value}</span>
      {subtitle && <span className="kpi-subtitle">{subtitle}</span>}
    </div>
  );
}
