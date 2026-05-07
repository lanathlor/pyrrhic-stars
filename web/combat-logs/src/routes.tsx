import { lazy } from "react";
import {
  createRouter,
  createRoute,
  createRootRouteWithContext,
  Outlet,
} from "@tanstack/react-router";
import type { QueryClient } from "@tanstack/react-query";
import { EncounterListPage } from "./components/encounters/EncounterListPage";
import { ReportPage } from "./components/report/ReportPage";
import { FightListPage } from "./components/fights/FightListPage";
import { FightPage } from "./components/fight/FightPage";
import { SummaryTab } from "./components/fight/tabs/SummaryTab";
import { DamageDoneTab } from "./components/fight/tabs/DamageDoneTab";
import { DamageTakenTab } from "./components/fight/tabs/DamageTakenTab";
import { HealingTab } from "./components/fight/tabs/HealingTab";
import { DeathsTab } from "./components/fight/tabs/DeathsTab";
import { TimelineTab } from "./components/fight/tabs/TimelineTab";
import { EventsTab } from "./components/fight/tabs/EventsTab";
import { fetchInstances, fetchInstance, fetchEvents } from "./api";

const TanStackRouterDevtools = import.meta.env.PROD
  ? () => null
  : lazy(() =>
      import("@tanstack/react-router-devtools").then((mod) => ({
        default: mod.TanStackRouterDevtools,
      }))
    );

interface RouterContext {
  queryClient: QueryClient;
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: () => (
    <>
      <header className="mb-8">
        <h1>Combat Logs</h1>
      </header>
      <Outlet />
      <TanStackRouterDevtools />
    </>
  ),
});

// Landing page: encounter overview (grouped sim runs)
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: EncounterListPage,
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData({
      queryKey: ["instances", { source: "simulation", limit: "10000" }],
      queryFn: () => fetchInstances({ source: "simulation", limit: "10000" }),
    }),
});

// Aggregate report for a specific encounter
const reportRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/report/$encounterId",
  component: ReportPage,
  loader: ({ params: { encounterId }, context: { queryClient } }) =>
    queryClient.ensureQueryData({
      queryKey: ["instances", { encounter_id: encounterId, source: "simulation", limit: "10000" }],
      queryFn: () => fetchInstances({ encounter_id: encounterId, source: "simulation", limit: "10000" }),
    }),
});

// Original fight list (all logs, live + sim)
const fightsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/fights",
  component: FightListPage,
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData({
      queryKey: ["instances", {}],
      queryFn: () => fetchInstances(),
    }),
});

// Individual fight detail
const fightRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/fight/$instanceId",
  component: FightPage,
  loader: ({ params: { instanceId }, context: { queryClient } }) => {
    queryClient.ensureQueryData({
      queryKey: ["instance", instanceId],
      queryFn: () => fetchInstance(instanceId),
    });
    queryClient.ensureQueryData({
      queryKey: ["events", instanceId],
      queryFn: () => fetchEvents(instanceId),
    });
  },
});

const summaryRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/",
  component: SummaryTab,
});

const damageRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/damage",
  component: DamageDoneTab,
});

const takenRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/taken",
  component: DamageTakenTab,
});

const healingRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/healing",
  component: HealingTab,
});

const deathsRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/deaths",
  component: DeathsTab,
});

const timelineRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/timeline",
  component: TimelineTab,
});

const eventsRoute = createRoute({
  getParentRoute: () => fightRoute,
  path: "/events",
  component: EventsTab,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  reportRoute,
  fightsRoute,
  fightRoute.addChildren([
    summaryRoute,
    damageRoute,
    takenRoute,
    healingRoute,
    deathsRoute,
    timelineRoute,
    eventsRoute,
  ]),
]);

export const router = createRouter({
  routeTree,
  context: { queryClient: undefined! }, // provided at render time
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
