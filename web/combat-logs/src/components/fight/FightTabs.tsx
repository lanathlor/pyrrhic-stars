import { Link, useMatchRoute } from "@tanstack/react-router";

interface Props {
  instanceId: string;
}

const TABS = [
  { path: "/fight/$instanceId", label: "Summary" },
  { path: "/fight/$instanceId/damage", label: "Damage Done" },
  { path: "/fight/$instanceId/taken", label: "Damage Taken" },
  { path: "/fight/$instanceId/healing", label: "Healing" },
  { path: "/fight/$instanceId/deaths", label: "Deaths" },
  { path: "/fight/$instanceId/timeline", label: "Timeline" },
  { path: "/fight/$instanceId/events", label: "Events" },
] as const;

const tabBase = "px-4 py-2.5 text-sm text-text-muted border-b-2 border-transparent transition-colors hover:text-text no-underline whitespace-nowrap";
const tabActive = "!text-accent !border-b-accent";

export function FightTabs({ instanceId }: Props) {
  const matchRoute = useMatchRoute();

  return (
    <nav className="flex flex-1">
      {TABS.map((tab) => {
        const isActive = !!matchRoute({
          to: tab.path,
          params: { instanceId },
          fuzzy: false,
        });
        return (
          <Link
            key={tab.path}
            to={tab.path}
            params={{ instanceId }}
            replace
            className={`${tabBase} ${isActive ? tabActive : ""}`}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
