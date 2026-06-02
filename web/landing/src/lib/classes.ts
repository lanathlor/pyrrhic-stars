// 6 classes. Mirrors docs/design/classes/README.md. Keep in sync.
//
// Accent colors are deliberately muted (per docs/design/ui-language.md:
// "restrained color usage"). One controlled accent per class.

export interface ClassEntry {
  slug: string;
  name: string;
  genre: string;
  identity: string;
  /** Tailwind 4 token name without the `--color-class-` prefix. */
  accent: ClassAccent;
}

export type ClassAccent =
  | "gunner"
  | "vanguard"
  | "arcanotechnicien"
  | "engineer"
  | "blade-dancer"
  | "tutelaire";

export const CLASSES: ClassEntry[] = [
  {
    slug: "gunner",
    name: "Gunner",
    genre: "First-Person Shooter",
    identity: "Aim, reposition, pressure. Three specs: assault, marksman, area denial.",
    accent: "gunner",
  },
  {
    slug: "vanguard",
    name: "Vanguard",
    genre: "Souls-like Action Melee",
    identity: "Read, commit, punish. Blade swirl, directional block, flanking pressure.",
    accent: "vanguard",
  },
  {
    slug: "arcanotechnicien",
    name: "Arcanotechnicien",
    genre: "Tactical Channeling",
    identity: "Long commits, huge payoffs. Destroyer, weaving battlemage, healing zones.",
    accent: "arcanotechnicien",
  },
  {
    slug: "engineer",
    name: "Engineer",
    genre: "Deployable Management",
    identity: "Place, pilot, disrupt. Turrets, drones, EMP fields.",
    accent: "engineer",
  },
  {
    slug: "blade-dancer",
    name: "Blade Dancer",
    genre: "Positional State Machine",
    identity: "Configs, GCDs, combos. Two blades for burst, six for sustained AoE.",
    accent: "blade-dancer",
  },
  {
    slug: "tutelaire",
    name: "Tutelaire",
    genre: "Aura Positioning",
    identity: "Stand here. Don't move. Aura tanking, ticking retribution, channelled healing.",
    accent: "tutelaire",
  },
];
