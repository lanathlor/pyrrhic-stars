import { useFightContext } from "../../../hooks/FightContext";
import { formatAmount, formatDps, formatDuration } from "../../../analysis/format";
import { KPICard } from "../../shared/KPICard";
import { DamageBar } from "../../shared/DamageBar";

export function SummaryTab() {
  const { analysis } = useFightContext();
  const { summary, damageDone, healingDone, deaths } = analysis;

  const maxDamage = damageDone[0]?.totalDamage ?? 1;
  const maxHealing = healingDone[0]?.totalHealing ?? 1;

  return (
    <div className="min-h-[200px]">
      <div className="grid grid-cols-[repeat(auto-fit,minmax(140px,1fr))] gap-3 mb-6">
        <KPICard label="Total Damage" value={formatAmount(summary.totalDamage)} />
        <KPICard label="Raid DPS" value={formatDps(summary.raidDps)} />
        <KPICard label="Total Healing" value={formatAmount(summary.totalHealing)} />
        <KPICard label="Deaths" value={summary.deathCount.toString()} />
        <KPICard label="Duration" value={formatDuration(summary.fightDurationMs)} />
        <KPICard label="Players" value={summary.playerCount.toString()} />
      </div>

      {damageDone.length > 0 && (
        <section className="mb-8">
          <h3>DPS Rankings</h3>
          <div className="flex flex-col gap-1">
            {damageDone.map((d) => (
              <DamageBar
                key={d.entityId}
                name={d.name}
                className={d.className}
                value={formatAmount(d.totalDamage)}
                secondary={`${formatDps(d.dps)} DPS`}
                percent={d.totalDamage / maxDamage}
              />
            ))}
          </div>
        </section>
      )}

      {healingDone.length > 0 && (
        <section className="mb-8">
          <h3>Healing Rankings</h3>
          <div className="flex flex-col gap-1">
            {healingDone.map((h) => (
              <DamageBar
                key={h.entityId}
                name={h.name}
                className={h.className}
                value={formatAmount(h.totalHealing)}
                secondary={`${formatDps(h.hps)} HPS`}
                percent={h.totalHealing / maxHealing}
              />
            ))}
          </div>
        </section>
      )}

      {deaths.length > 0 && (
        <section className="mb-8">
          <h3>Deaths ({deaths.length})</h3>
          <div className="flex flex-col gap-1">
            {deaths.map((d, i) => (
              <div key={i} className="flex items-center gap-3 text-sm py-1">
                <span className="text-text-muted tabular-nums min-w-12">{formatDuration(d.timestampMs)}</span>
                <span className="text-warning">{d.victimName}</span>
                {d.killingBlow && (
                  <span className="text-text-muted">
                    killed by {d.killingBlow.source} ({d.killingBlow.ability_id || "auto"})
                  </span>
                )}
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
