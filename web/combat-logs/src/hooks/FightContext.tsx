import { createContext, useContext, useState, type ReactNode } from "react";
import type { InstanceLog, LogEntry } from "../types";
import { useFightAnalysis } from "./useFightAnalysis";

interface FightContextValue {
  instance: InstanceLog;
  events: LogEntry[];
  analysis: ReturnType<typeof useFightAnalysis>;
  selectedPhase: string | null;
  setSelectedPhase: (phase: string | null) => void;
  selectedPlayer: string | null;
  setSelectedPlayer: (entityId: string | null) => void;
}

const FightCtx = createContext<FightContextValue | null>(null);

export function useFightContext(): FightContextValue {
  const ctx = useContext(FightCtx);
  if (!ctx) throw new Error("useFightContext must be used within FightProvider");
  return ctx;
}

interface FightProviderProps {
  instance: InstanceLog;
  events: LogEntry[];
  children: ReactNode;
}

export function FightProvider({ instance, events, children }: FightProviderProps) {
  const [selectedPhase, setSelectedPhase] = useState<string | null>(null);
  const [selectedPlayer, setSelectedPlayer] = useState<string | null>(null);
  const analysis = useFightAnalysis(instance, events, selectedPhase);

  return (
    <FightCtx.Provider
      value={{
        instance,
        events,
        analysis,
        selectedPhase,
        setSelectedPhase,
        selectedPlayer,
        setSelectedPlayer,
      }}
    >
      {children}
    </FightCtx.Provider>
  );
}
