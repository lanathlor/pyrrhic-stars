// Public roadmap data. Single source of truth for both the homepage Roadmap
// section and the full /roadmap page. Wording is paraphrased from
// docs/project/phases.md (which in turn defers phase order/shape to this site).
//
// Rules (see web/landing/CLAUDE.md):
// - No dates, no week-estimates. They go stale and look untrustworthy.
// - Phase 0.5 (post-clip validation) is internal-only. Do not surface it here.
// - Keep it public-facing: goals and deliverables, not internal tech stack.

export type PhaseStatus = "done" | "now" | "next" | "later";

export interface Phase {
  /** Short tag, e.g. "Phase 0". */
  label: string;
  /** Phase name, e.g. "Online alpha". */
  title: string;
  status: PhaseStatus;
  /** One-line goal for the phase. */
  goal: string;
  /** A short paragraph explaining the phase. Used on the /roadmap page. */
  summary: string;
  /** Deliverables. Used on the homepage section and the /roadmap page. */
  bullets: string[];
}

export const PHASES: Phase[] = [
  {
    label: "Phase 0",
    title: "Online alpha",
    status: "done",
    goal: "Five players, a hub, a class, a dungeon, one real boss.",
    summary:
      "The proof of concept. Five players meet in a hub, pick a class, group up, walk through a portal into a dungeon, fight one real boss, and leave. Everything runs server-authoritative over the internet, not on a LAN. This is done: four classes are playable across five specs, the combat systems resolve damage and cooldowns on the server, and the first dungeon is live with its first boss.",
    bullets: [
      "Server-authoritative online co-op",
      "Four classes, five specs playable",
      "Combat systems: damage, cooldowns, abilities",
      "First dungeon playable (1 boss live)",
    ],
  },
  {
    label: "Phase 1",
    title: "Content and economy",
    status: "done",
    goal: "Make the dungeon worth running: feel, loot, and stakes.",
    summary:
      "Everything that turns the alpha's single room into a dungeon you actually want to run. Trash packs lead into the boss, the full clear loop runs start to finish against a timer, and the combat-feel pass landed: telegraphs you read and animations that sell every hit. On top of that the reward loop is in: clear a run, earn mercenary scrip, and spend it at the merchant on gear. Difficulty modifiers (Overflux) let you raise the stakes for better payouts. All shipped.",
    bullets: [
      "Trash packs and full clear loop",
      "Combat feel: telegraphs and animations",
      "Earn scrip, buy gear from the merchant",
      "Difficulty modifiers (Overflux)",
    ],
  },
  {
    label: "Phase 2",
    title: "More bosses and depth",
    status: "now",
    goal: "Finish the first dungeon's gauntlet of bosses.",
    summary:
      "The work in front of me right now. The first dungeon still leans on a single boss, so the focus is more encounters: additional bosses, a full multi-boss clear from the first trash pack to the last, and deeper mechanics that each fight teaches. This is where the active work is going.",
    bullets: [
      "More bosses for the first dungeon",
      "A full multi-boss clear, start to finish",
      "Deeper encounter mechanics",
    ],
  },
  {
    label: "Phase 3",
    title: "Polish",
    status: "later",
    goal: "Round out the roster and open up the world.",
    summary:
      "Rounding everything out. The last two classes (the Engineer and the Tutelaire) bring the roster to six, and a group finder means you no longer need a pre-made team to play. The world opens up too: a first open-world zone and an adventuring loop to explore it, beyond the dungeon runs. Beyond this lies a longer vision, but I will not promise what I have not started.",
    bullets: [
      "Last two classes (Engineer, Tutelaire)",
      "Group finder",
      "First open-world zone",
      "Adventuring loop",
      "Next phases...",
    ],
  },
];
