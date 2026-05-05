import { Routes, Route } from "react-router-dom";
import { FightListPage } from "./components/fights/FightListPage";
import { FightPage } from "./components/fight/FightPage";
import { SummaryTab } from "./components/fight/tabs/SummaryTab";
import { DamageDoneTab } from "./components/fight/tabs/DamageDoneTab";
import { DamageTakenTab } from "./components/fight/tabs/DamageTakenTab";
import { HealingTab } from "./components/fight/tabs/HealingTab";
import { DeathsTab } from "./components/fight/tabs/DeathsTab";
import { TimelineTab } from "./components/fight/tabs/TimelineTab";
import { EventsTab } from "./components/fight/tabs/EventsTab";
import "./App.css";

function App() {
  return (
    <>
      <header>
        <h1>Combat Logs</h1>
      </header>
      <Routes>
        <Route path="/" element={<FightListPage />} />
        <Route path="/fight/:instanceId" element={<FightPage />}>
          <Route index element={<SummaryTab />} />
          <Route path="damage" element={<DamageDoneTab />} />
          <Route path="taken" element={<DamageTakenTab />} />
          <Route path="healing" element={<HealingTab />} />
          <Route path="deaths" element={<DeathsTab />} />
          <Route path="timeline" element={<TimelineTab />} />
          <Route path="events" element={<EventsTab />} />
        </Route>
      </Routes>
    </>
  );
}

export default App;
