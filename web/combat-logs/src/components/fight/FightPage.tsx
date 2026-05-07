import { Outlet, useParams } from "@tanstack/react-router";
import { useFightData } from "../../hooks/useFightData";
import { FightProvider } from "../../hooks/FightContext";
import { FightHeader } from "./FightHeader";
import { FightTabsBar } from "./FightTabsBar";
import { BackLink } from "../shared/BackLink";

export function FightPage() {
  const { instanceId } = useParams({ from: "/fight/$instanceId" });
  const { instance, events, loading, error } = useFightData(instanceId);

  if (loading) return <p className="text-text-muted">Loading fight data...</p>;
  if (error) return <p className="text-danger">{error}</p>;
  if (!instance) return null;

  return (
    <FightProvider instance={instance} events={events}>
      <div>
        <BackLink to={`/report/${instance.encounter_id}`} />
        <FightHeader instance={instance} />
        <FightTabsBar instanceId={instanceId} />
        <Outlet />
      </div>
    </FightProvider>
  );
}
