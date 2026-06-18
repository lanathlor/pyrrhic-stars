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
    goal: "Prove the core loop works online.",
    summary:
      "The proof of concept, and it is done. Five people meet in a hub, pick a class, group up, and head through a portal into a dungeon to fight a boss. It all runs on a server over the internet, so you can actually play it with friends who are not in the room. There are four playable classes with five specs between them, the server handles the damage and the cooldowns, and the first dungeon has its first boss.",
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
    goal: "Turn the demo room into a dungeon worth replaying.",
    summary:
      "This is what pushed the alpha past a single empty room, and it is all in the build now. You fight through packs of enemies to reach the boss, and the whole run is played against a clear timer. Combat got a proper feel pass, with telegraphs you can read and animations behind every hit. The reward loop works too. Finish a run, earn mercenary scrip, take it to the merchant for gear. And if you want a harder run for a better payout, the Overflux modifiers let you crank things up.",
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
    goal: "Give the first dungeon more than one boss.",
    summary:
      "This is the part I am working on now. The dungeon still hangs on a single boss, so it needs more of them, with the fights in between, so that a full clear takes you all the way from the first pack of enemies to the last. The new encounters are also where I get to build deeper, stranger mechanics.",
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
    goal: "Fill out the roster and step outside the dungeon.",
    summary:
      "The later stuff. The last two classes, the Engineer and the Tutelaire, bring the roster up to six. A group finder means you do not have to bring your own team to play. And the game starts to leave the dungeon behind, with a first open-world zone and a reason to go wander around in it. There is more I want to do past that, but I would rather not promise things I have not started yet.",
    bullets: [
      "Last two classes (Engineer, Tutelaire)",
      "Group finder",
      "First open-world zone",
      "Adventuring loop",
      "Next phases...",
    ],
  },
];
