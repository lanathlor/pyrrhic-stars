interface Props {
  encounter: string;
  setEncounter: (v: string) => void;
  outcome: string;
  setOutcome: (v: string) => void;
  source: string;
  setSource: (v: string) => void;
}

export function FightListFilters({ encounter, setEncounter, outcome, setOutcome, source, setSource }: Props) {
  return (
    <div className="filter-bar">
      <input
        type="text"
        placeholder="Encounter name..."
        value={encounter}
        onChange={(e) => setEncounter(e.target.value)}
        className="filter-input"
      />
      <select value={outcome} onChange={(e) => setOutcome(e.target.value)} className="filter-select">
        <option value="">All Outcomes</option>
        <option value="player_win">Kill</option>
        <option value="boss_win">Wipe</option>
        <option value="timeout">Timeout</option>
      </select>
      <select value={source} onChange={(e) => setSource(e.target.value)} className="filter-select">
        <option value="">All Sources</option>
        <option value="live">Live</option>
        <option value="simulation">Simulation</option>
      </select>
    </div>
  );
}
