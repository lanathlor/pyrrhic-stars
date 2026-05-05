import { NavLink } from "react-router-dom";

interface Props {
  instanceId: string;
}

const TABS = [
  { path: "", label: "Summary" },
  { path: "damage", label: "Damage Done" },
  { path: "taken", label: "Damage Taken" },
  { path: "healing", label: "Healing" },
  { path: "deaths", label: "Deaths" },
  { path: "timeline", label: "Timeline" },
  { path: "events", label: "Events" },
];

export function FightTabs({ instanceId }: Props) {
  const base = `/fight/${instanceId}`;
  return (
    <nav className="tabs">
      {TABS.map((tab) => (
        <NavLink
          key={tab.path}
          to={tab.path ? `${base}/${tab.path}` : base}
          end={tab.path === ""}
          className={({ isActive }) => `tab ${isActive ? "tab-active" : ""}`}
        >
          {tab.label}
        </NavLink>
      ))}
    </nav>
  );
}
