import { useParams, Outlet, useNavigate } from "react-router-dom";
import { useFightData } from "../../hooks/useFightData";
import { FightProvider } from "../../hooks/FightContext";
import { FightHeader } from "./FightHeader";
import { FightTabsBar } from "./FightTabsBar";

export function FightPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const navigate = useNavigate();
  const { instance, events, loading, error } = useFightData(instanceId!);

  if (loading) return <p style={{ color: "var(--text-muted)" }}>Loading fight data...</p>;
  if (error) return <p style={{ color: "var(--danger)" }}>{error}</p>;
  if (!instance) return null;

  return (
    <FightProvider instance={instance} events={events}>
      <div>
        <button onClick={() => navigate("/")} className="btn" style={{ marginBottom: "1rem" }}>
          Back to Fights
        </button>
        <FightHeader instance={instance} />
        <FightTabsBar instanceId={instanceId!} />
        <Outlet />
      </div>
    </FightProvider>
  );
}
