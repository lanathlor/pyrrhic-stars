export const CLASS_COLORS: Record<string, string> = {
  gunner: "#e6cc80",
  vanguard: "#3b82f6",
  blade_dancer: "#a855f7",
};

export const CLASS_DISPLAY_NAMES: Record<string, string> = {
  gunner: "Gunner",
  vanguard: "Vanguard",
  blade_dancer: "Blade Dancer",
};

export const EVENT_TYPES = {
  DAMAGE: 1,
  HEAL: 2,
  BUFF_APPLY: 3,
  BUFF_REMOVE: 4,
  BUFF_TICK: 5,
  CAST_START: 6,
  CAST_END: 7,
  CD_START: 8,
  CD_END: 9,
  DODGE: 10,
  DEATH: 11,
  PHASE_CHANGE: 12,
} as const;
