interface Props {
  encounter: string;
  setEncounter: (v: string) => void;
  outcome: string;
  setOutcome: (v: string) => void;
  source: string;
  setSource: (v: string) => void;
}

const inputCls = "px-3 py-1.5 bg-surface border border-border rounded text-text text-sm min-w-[150px] placeholder:text-text-muted";
const selectCls = "px-3 py-1.5 bg-surface border border-border rounded text-text text-sm";

export function FightListFilters({ encounter, setEncounter, outcome, setOutcome, source, setSource }: Props) {
  return (
    <div className="flex flex-col gap-3 mb-4">
      <input
        type="text"
        placeholder="Encounter name..."
        value={encounter}
        onChange={(e) => setEncounter(e.target.value)}
        className={inputCls}
      />
      <select value={outcome} onChange={(e) => setOutcome(e.target.value)} className={selectCls}>
        <option value="">All Outcomes</option>
        <option value="player_win">Kill</option>
        <option value="boss_win">Wipe</option>
        <option value="timeout">Timeout</option>
      </select>
      <select value={source} onChange={(e) => setSource(e.target.value)} className={selectCls}>
        <option value="">All Sources</option>
        <option value="live">Live</option>
        <option value="simulation">Simulation</option>
      </select>
    </div>
  );
}
