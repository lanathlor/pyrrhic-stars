import { useMemo, useState } from "react";
import { useParams } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { fetchInstances, fetchEncounterStats } from "../../api";
import {
  computeReportOverview,
  computeCompStats,
  computeClassDPSStats,
  computeClassHPSStats,
  computeDeathStats,
  computeRaidDPSStats,
  computePhaseReach,
  computeWipePhases,
  computeProfileCombatStats,
} from "../../analysis/report";
import { BackLink } from "../shared/BackLink";
import { ReportHeader } from "./ReportHeader";
import { OutcomeChart } from "./OutcomeChart";
import { DurationHistogram } from "./DurationHistogram";
import { ClassDPSChart } from "./ClassDPSChart";
import { BossAbilityTable } from "./BossAbilityTable";
import { DeathStats } from "./DeathStats";
import { PhaseAnalysis } from "./PhaseAnalysis";
import { ProfileCombatCards } from "./ProfileCombatCards";
import { CompTable } from "./CompTable";
import { RunsTable } from "./RunsTable";

export function ReportPage() {
  const { encounterId } = useParams({ from: "/report/$encounterId" });
  const [percentileMode, setPercentileMode] = useState<"p95" | "p99">("p95");

  const { data: instances = [], isLoading, error } = useQuery({
    queryKey: ["instances", { encounter_id: encounterId, source: "simulation", limit: "10000" }],
    queryFn: () => fetchInstances({ encounter_id: encounterId, source: "simulation", limit: "10000" }),
  });

  const { data: encounterStats } = useQuery({
    queryKey: ["encounterStats", encounterId],
    queryFn: () => fetchEncounterStats(encounterId, { source: "simulation" }),
    enabled: instances.length > 0,
  });

  const overview = useMemo(
    () => (instances.length > 0 ? computeReportOverview(instances) : null),
    [instances]
  );
  const compStats = useMemo(() => computeCompStats(instances), [instances]);

  // Combat aggregate stats (depends on encounterStats being loaded)
  const classDPS = useMemo(
    () => (encounterStats ? computeClassDPSStats(encounterStats, instances) : []),
    [encounterStats, instances]
  );
  const classHPS = useMemo(
    () => (encounterStats ? computeClassHPSStats(encounterStats, instances) : []),
    [encounterStats, instances]
  );
  const deathStats = useMemo(
    () => (encounterStats ? computeDeathStats(encounterStats, instances) : null),
    [encounterStats, instances]
  );
  const raidDPSStats = useMemo(
    () => (encounterStats ? computeRaidDPSStats(encounterStats, instances) : null),
    [encounterStats, instances]
  );
  const phaseReach = useMemo(
    () => (encounterStats ? computePhaseReach(encounterStats, instances) : []),
    [encounterStats, instances]
  );
  const wipePhases = useMemo(
    () => (encounterStats ? computeWipePhases(encounterStats, instances) : []),
    [encounterStats, instances]
  );
  const profileCombat = useMemo(
    () => (encounterStats ? computeProfileCombatStats(encounterStats, instances) : []),
    [encounterStats, instances]
  );

  if (isLoading) return <p className="text-text-muted">Loading report...</p>;
  if (error) return <p className="text-danger">{error.message}</p>;
  if (!overview) return <p className="text-text-muted">No simulation data for this encounter.</p>;

  return (
    <div>
      <BackLink to="/" />
      <ReportHeader
        overview={overview}
        raidDPSStats={raidDPSStats}
        deathStats={deathStats}
        percentileMode={percentileMode}
      />

      {/* Duration + Outcome */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_300px] gap-6 mb-8">
        <DurationHistogram
          stats={overview.durationStats}
          percentileMode={percentileMode}
          setPercentileMode={setPercentileMode}
        />
        <OutcomeChart
          wins={overview.wins}
          losses={overview.losses}
          timeouts={overview.timeouts}
        />
      </div>

      {/* Class DPS + HPS */}
      {classDPS.length > 0 && (
        <section className="mb-8">
          <ClassDPSChart
            distributions={classDPS}
            percentileMode={percentileMode}
            label="DPS by Class"
          />
        </section>
      )}
      {classHPS.length > 0 && (
        <section className="mb-8">
          <ClassDPSChart
            distributions={classHPS}
            percentileMode={percentileMode}
            label="HPS by Class"
          />
        </section>
      )}

      {/* Boss Abilities */}
      {encounterStats && encounterStats.boss_abilities.length > 0 && (
        <section className="mb-8">
          <BossAbilityTable abilities={encounterStats.boss_abilities} />
        </section>
      )}

      {/* Deaths + Phases */}
      {(deathStats || phaseReach.length > 0) && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          {deathStats && (
            <DeathStats
              deathStats={deathStats}
              bossAbilities={encounterStats?.boss_abilities ?? []}
              percentileMode={percentileMode}
            />
          )}
          {phaseReach.length > 0 && (
            <PhaseAnalysis phaseReach={phaseReach} wipePhases={wipePhases} />
          )}
        </div>
      )}

      {/* Per-Profile Breakdown */}
      {profileCombat.length > 1 && (
        <section className="mb-8">
          <h3>Per-Profile Breakdown</h3>
          <ProfileCombatCards profiles={profileCombat} percentileMode={percentileMode} />
        </section>
      )}

      {/* Compositions */}
      {compStats.length > 1 && (
        <section className="mb-8">
          <h3>Compositions</h3>
          <CompTable comps={compStats} percentileMode={percentileMode} />
        </section>
      )}

      {/* All Runs */}
      <section className="mb-8">
        <h3>All Runs ({instances.length})</h3>
        <RunsTable instances={instances} />
      </section>
    </div>
  );
}
