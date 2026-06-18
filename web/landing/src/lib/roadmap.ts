// Public roadmap data. Single source of truth for both the homepage Roadmap
// section and the full /roadmap page. Wording is paraphrased from
// docs/project/phases.md (which in turn defers phase order/shape to this site).
//
// Rules (see web/landing/CLAUDE.md):
// - No dates, no week-estimates. They go stale and look untrustworthy.
// - Phase 0.5 (post-clip validation) is internal-only. Do not surface it here.
// - Keep it public-facing: goals and deliverables, not internal tech stack.
//
// Locale-neutral fields (id, label, status) live on PHASE_BASE; the translatable
// title/goal/summary/bullets live in PHASE_TEXT keyed by locale. Use
// getRoadmap(lang) to get the merged, ordered phases.

import { defaultLang, type Lang } from "../i18n/utils";

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

interface PhaseBase {
  id: string;
  label: string;
  status: PhaseStatus;
}

const PHASE_BASE: PhaseBase[] = [
  { id: "phase0", label: "Phase 0", status: "done" },
  { id: "phase1", label: "Phase 1", status: "done" },
  { id: "phase2", label: "Phase 2", status: "now" },
  { id: "phase3", label: "Phase 3", status: "later" },
];

type PhaseText = Pick<Phase, "title" | "goal" | "summary" | "bullets">;

const PHASE_TEXT: Record<Lang, Record<string, PhaseText>> = {
  en: {
    phase0: {
      title: "Online alpha",
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
    phase1: {
      title: "Content and economy",
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
    phase2: {
      title: "More bosses and depth",
      goal: "Give the first dungeon more than one boss.",
      summary:
        "This is the part I am working on now. The dungeon still hangs on a single boss, so it needs more of them, with the fights in between, so that a full clear takes you all the way from the first pack of enemies to the last. The new encounters are also where I get to build deeper, stranger mechanics.",
      bullets: [
        "More bosses for the first dungeon",
        "A full multi-boss clear, start to finish",
        "Deeper encounter mechanics",
      ],
    },
    phase3: {
      title: "Polish",
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
  },
  fr: {
    phase0: {
      title: "Alpha en ligne",
      goal: "Prouver que la boucle de jeu fonctionne en ligne.",
      summary:
        "La preuve de concept, et elle est faite. Cinq personnes se retrouvent dans un hub, choisissent une classe, forment un groupe et franchissent un portail vers un donjon pour affronter un boss. Tout tourne sur un serveur via internet, vous pouvez donc vraiment y jouer avec des amis qui ne sont pas dans la même pièce. Il y a quatre classes jouables pour cinq spécialisations au total, le serveur gère les dégâts et les temps de recharge, et le premier donjon a son premier boss.",
      bullets: [
        "Coopération en ligne, autorité serveur",
        "Quatre classes, cinq spécialisations jouables",
        "Systèmes de combat : dégâts, recharges, capacités",
        "Premier donjon jouable (1 boss en place)",
      ],
    },
    phase1: {
      title: "Contenu et économie",
      goal: "Transformer la salle de démo en un donjon qui vaut le coup d'être rejoué.",
      summary:
        "C'est ce qui a fait sortir l'alpha d'une simple salle vide, et tout est dans la version jouable maintenant. Vous combattez des groupes d'ennemis pour atteindre le boss, et toute la partie se joue contre un chrono de clôture. Le combat a eu une vraie passe de ressenti, avec des télégraphes lisibles et des animations derrière chaque coup. La boucle de récompense fonctionne aussi. Terminez une partie, gagnez de la solde de mercenaire, dépensez-la chez le marchand pour de l'équipement. Et si vous voulez une partie plus difficile pour un meilleur gain, les modificateurs Surflux vous laissent monter la pression.",
      bullets: [
        "Groupes d'ennemis et boucle de clôture complète",
        "Ressenti du combat : télégraphes et animations",
        "Gagnez de la solde, achetez de l'équipement chez le marchand",
        "Modificateurs de difficulté (Surflux)",
      ],
    },
    phase2: {
      title: "Plus de boss et de profondeur",
      goal: "Donner plus d'un boss au premier donjon.",
      summary:
        "C'est la partie sur laquelle je travaille en ce moment. Le donjon ne tient encore que sur un seul boss, il lui en faut donc davantage, avec les combats entre eux, pour qu'une clôture complète vous mène du premier groupe d'ennemis au dernier. Les nouvelles rencontres sont aussi là où je peux construire des mécaniques plus profondes et plus étranges.",
      bullets: [
        "Plus de boss pour le premier donjon",
        "Une clôture complète multi-boss, du début à la fin",
        "Des mécaniques de rencontre plus profondes",
      ],
    },
    phase3: {
      title: "Peaufinage",
      goal: "Compléter le roster et sortir du donjon.",
      summary:
        "Les éléments plus tardifs. Les deux dernières classes, l'Ingénieur et le Tutelaire, portent le roster à six. Un outil de recherche de groupe signifie que vous n'avez pas à amener votre propre équipe pour jouer. Et le jeu commence à quitter le donjon, avec une première zone en monde ouvert et une raison d'aller s'y promener. Il y a plus que je veux faire au-delà, mais je préfère ne pas promettre des choses que je n'ai pas commencées.",
      bullets: [
        "Les deux dernières classes (Ingénieur, Tutelaire)",
        "Recherche de groupe",
        "Première zone en monde ouvert",
        "Boucle d'aventure",
        "Phases suivantes...",
      ],
    },
  },
};

/** The full roadmap, ordered, with text resolved for `lang`. */
export function getRoadmap(lang: Lang): Phase[] {
  const text = PHASE_TEXT[lang] ?? PHASE_TEXT[defaultLang];
  return PHASE_BASE.map((base) => ({
    label: base.label,
    status: base.status,
    ...text[base.id],
  }));
}
