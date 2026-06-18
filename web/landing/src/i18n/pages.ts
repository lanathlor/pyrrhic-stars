// Longer prose blocks, keyed by locale: the four homepage feature pillars and
// the About / Roadmap / Contribute page copy. Kept out of ./ui.ts because these
// are paragraphs, not labels. Page body components in src/components/pages/ read
// from here via getContent(lang).
//
// Constraints (web/landing/CLAUDE.md): never the word "MMO"; feature pillars
// stay factually consistent with docs/design/combat.md but in plain voice;
// roadmap copy carries no dates or week-estimates.

import { defaultLang, type Lang } from "./ui";

export interface Pillar {
  title: string;
  body: string;
  tags?: string[];
}

export interface NamedItem {
  name: string;
  desc: string;
}

export interface PageContent {
  features: Pillar[];
  about: {
    metaTitle: string;
    metaDescription: string;
    eyebrow: string;
    heading: string;
    p1: string;
    p2: string;
    /** Paragraph 3 wraps an inline "open source" link to the repo. */
    p3before: string;
    p3link: string;
    p4: string;
    tracksHeading: string;
    tracksIntro: string;
    tracks: NamedItem[];
    tracksClosing: string;
    rewardsHeading: string;
    rewardsBody: string;
  };
  roadmap: {
    metaTitle: string;
    metaDescription: string;
    intro: string;
    footerBefore: string;
    footerLink: string;
    footerAfter: string;
  };
  contribute: {
    metaTitle: string;
    metaDescription: string;
    eyebrow: string;
    heading: string;
    intro: string;
    needHeading: string;
    needs: NamedItem[];
    startHeading: string;
    startBefore: string;
    startLink: string;
    startAfter: string;
    startNoRepo: string;
    touchHeading: string;
    touchBefore: string;
    touchLink: string;
    touchAfter: string;
    touchNoDiscord: string;
  };
}

const content: Record<Lang, PageContent> = {
  en: {
    features: [
      {
        title: "Six classes, six different genres.",
        body: "Each class has its own camera, its own controls, and a combat loop that has little to do with the others. They are still all in the same dungeon, on the same boss, at the same time.",
        tags: ["FPS", "Souls-like", "Channeler", "Deployables", "Blades", "Auras"],
      },
      {
        title: "Skill-based combat, no threat tables.",
        body: "A boss never chips you with damage you had no way to avoid. It winds up, you see it coming, and you move. The Shield Vanguard holds a piece of the arena by standing in it; nothing in the game forces a boss to attack one player over another.",
      },
      {
        title: "Five-player co-op, no fixed roles.",
        body: "There are healers and tanks, and they are good at what they do. You just are not required to bring one. With enough skill a group can clear without them.",
      },
      {
        title: "Built in public.",
        body: "I try to put out something real on a regular basis: a new system, a bit of footage, whatever actually works that week. No mockups, and no promises about things I have not built yet.",
      },
    ],
    about: {
      metaTitle: "About",
      metaDescription: "About Pyrrhic Stars and the person building it.",
      eyebrow: "About",
      heading: "About Pyrrhic Stars.",
      p1: "Pyrrhic Stars is a co-op action game where every class plays a different genre. Five of you take on the same dungeon and the same boss, but what you are actually doing depends on the class you picked. The Gunner is playing a first-person shooter. The Vanguard is in a Souls-like, reading the boss for an opening. The Blade Dancer is working through a state machine of blade configs. Same fight, and none of you are really playing the same game.",
      p2: "I have played co-op games for about twenty years, and they nearly all work the same way. You bring a tank, a healer, and some damage, and whatever class you picked, the job underneath was the same one: be in the right place and hit your buttons without dying. Your class decided which buttons. It almost never decided what the game actually felt like to play. That is the part I want to break.",
      p3before: "I build this on my own, mostly on weekends, on top of a day job. My background is backend development rather than games, which is why the server and the systems are further along than the art is right now. It is all ",
      p3link: "open source",
      p4: "The inspirations are not hard to spot. World of Warcraft, Furi, and the Souls games. There are two things I care about here: that it ends up fun, and that it stays open for people to dig into. It is very early still, an extremely rough early access of some sorts.",
      tracksHeading: "Progress tracks",
      tracksIntro: "In v1, the goal is four progress tracks, each its own way to play:",
      tracks: [
        { name: "Mercenary.", desc: "Mythic+ style dungeons." },
        { name: "Paragon.", desc: "Raids." },
        { name: "Hero.", desc: "Monster Hunter style: solo or small-group fights against one big boss." },
        { name: "Explorer.", desc: "Open-world survival." },
      ],
      tracksClosing: "Each track awards the gear that is best for that track. No need to run dungeons if you want to raid.",
      rewardsHeading: "How rewards work",
      rewardsBody: "No levels, no endless grind, no tedious fetch quests. And nothing drops on a dice roll. You clear something, it pays out in tokens, and you spend those on the gear you actually want.",
    },
    roadmap: {
      metaTitle: "Roadmap",
      metaDescription: "How Pyrrhic Stars gets built: phase by phase, each ending with something concrete. What's done, what's now, and what comes next.",
      intro: "I build this in phases. Each phase ends with something concrete: a clip, a playable build, a Steam page. Before the next one starts. No release dates, no features announced before I have started them. The repository is the source of truth; this page is the plan around it.",
      footerBefore: "Phases shift as I learn what the game wants to be. If something here does not match the build, the build wins and I have not updated this page yet. Follow along on the ",
      footerLink: "devlog",
      footerAfter: ".",
    },
    contribute: {
      metaTitle: "Contribute",
      metaDescription: "How to contribute to Pyrrhic Stars. The whole project is open source.",
      eyebrow: "Open source",
      heading: "How to contribute.",
      intro: "Anyone is welcome to contribute. The whole thing is open source, under the AGPL. I come from backend development, so the spots where I need help most are probably obvious to anyone who has played the build.",
      needHeading: "Where I need help most",
      needs: [
        { name: "Art and animation.", desc: "Graphics, good PBR materials, and especially rigging, meshes, and animation. This is my weakest area by far." },
        { name: "The world.", desc: "More nuance: NPCs, set dressing, detailing." },
        { name: "Sound and music.", desc: "Extremely welcome." },
        { name: "Code.", desc: "Debugging, advancing features, refactoring." },
        { name: "Feedback.", desc: "What feels good, what falls flat, what is confusing. Honestly the most useful thing at this stage." },
      ],
      startHeading: "Getting started",
      startBefore: "The code lives ",
      startLink: "on the public repository",
      startAfter: ". Open an issue or a pull request.",
      startNoRepo: "The public repository link is coming soon.",
      touchHeading: "Get in touch",
      touchBefore: "Say hi ",
      touchLink: "on Discord",
      touchAfter: ".",
      touchNoDiscord: "A Discord is coming soon.",
    },
  },

  fr: {
    features: [
      {
        title: "Six classes, six genres différents.",
        body: "Chaque classe a sa propre caméra, ses propres commandes et une boucle de combat qui n'a presque rien à voir avec les autres. Elles sont pourtant toutes dans le même donjon, sur le même boss, en même temps.",
        tags: ["FPS", "Souls-like", "Incantateur", "Déploiements", "Lames", "Auras"],
      },
      {
        title: "Un combat technique, sans table de menace.",
        body: "Un boss ne vous grignote jamais avec des dégâts impossibles à éviter. Il s'arme, vous le voyez venir, et vous bougez. L'Avant-garde au bouclier tient une portion de l'arène rien qu'en s'y tenant ; rien dans le jeu ne force un boss à attaquer un joueur plutôt qu'un autre.",
      },
      {
        title: "Coopération à cinq, sans rôles imposés.",
        body: "Il y a des soigneurs et des tanks, et ils sont bons dans leur rôle. Vous n'êtes simplement pas obligés d'en amener un. Avec assez de maîtrise, un groupe peut réussir sans eux.",
      },
      {
        title: "Développé en public.",
        body: "J'essaie de sortir quelque chose de réel à intervalle régulier : un nouveau système, un bout de séquence, ce qui marche vraiment cette semaine-là. Pas de maquettes, et aucune promesse sur ce que je n'ai pas encore construit.",
      },
    ],
    about: {
      metaTitle: "À propos",
      metaDescription: "À propos de Pyrrhic Stars et de la personne qui le développe.",
      eyebrow: "À propos",
      heading: "À propos de Pyrrhic Stars.",
      p1: "Pyrrhic Stars est un jeu d'action coopératif où chaque classe joue à un genre différent. Vous affrontez à cinq le même donjon et le même boss, mais ce que vous faites réellement dépend de la classe que vous avez choisie. Le Tireur joue à un jeu de tir à la première personne. L'Avant-garde est dans un Souls-like, à lire le boss pour saisir une ouverture. Le Danseur de lames navigue dans une machine à états de configurations de lames. Même combat, et aucun de vous ne joue vraiment au même jeu.",
      p2: "Je joue à des jeux coopératifs depuis une vingtaine d'années, et ils fonctionnent presque tous de la même manière. On amène un tank, un soigneur et des dégâts, et quelle que soit la classe choisie, le travail de fond restait le même : être au bon endroit et appuyer sur ses touches sans mourir. Votre classe décidait quelles touches. Elle ne décidait presque jamais de ce que le jeu donnait vraiment comme sensation. C'est cette partie-là que je veux casser.",
      p3before: "Je développe ce jeu seul, surtout le week-end, en plus d'un emploi à temps plein. Je viens du développement backend plutôt que du jeu vidéo, ce qui explique pourquoi le serveur et les systèmes sont plus avancés que les graphismes pour l'instant. Tout est ",
      p3link: "open source",
      p4: "Les inspirations ne sont pas difficiles à repérer. World of Warcraft, Furi et les jeux Souls. Deux choses me tiennent à cœur ici : que ce soit amusant au final, et que ça reste ouvert pour que les gens puissent y mettre les mains. C'est encore très tôt, une sorte d'accès anticipé extrêmement brut.",
      tracksHeading: "Voies de progression",
      tracksIntro: "En v1, l'objectif est quatre voies de progression, chacune sa propre façon de jouer :",
      tracks: [
        { name: "Mercenaire.", desc: "Donjons façon Mythique+." },
        { name: "Parangon.", desc: "Raids." },
        { name: "Héros.", desc: "Façon Monster Hunter : combats en solo ou en petit groupe contre un seul gros boss." },
        { name: "Explorateur.", desc: "Survie en monde ouvert." },
      ],
      tracksClosing: "Chaque voie récompense l'équipement le mieux adapté à cette voie. Pas besoin de faire des donjons si vous voulez raider.",
      rewardsHeading: "Comment fonctionnent les récompenses",
      rewardsBody: "Pas de niveaux, pas de farm sans fin, pas de quêtes fastidieuses. Et rien ne tombe sur un jet de dés. Vous réussissez quelque chose, ça paie en jetons, et vous les dépensez sur l'équipement que vous voulez vraiment.",
    },
    roadmap: {
      metaTitle: "Feuille de route",
      metaDescription: "Comment Pyrrhic Stars se construit : phase par phase, chacune se terminant par quelque chose de concret. Ce qui est fait, ce qui est en cours, et ce qui vient ensuite.",
      intro: "Je construis ce jeu par phases. Chaque phase se termine par quelque chose de concret : un clip, une version jouable, une page Steam. Avant que la suivante ne commence. Pas de dates de sortie, pas de fonctionnalités annoncées avant de les avoir commencées. Le dépôt est la source de vérité ; cette page est le plan qui l'entoure.",
      footerBefore: "Les phases évoluent à mesure que je découvre ce que le jeu veut devenir. Si quelque chose ici ne correspond pas à la version jouable, c'est la version qui gagne et c'est que je n'ai pas encore mis cette page à jour. Suivez le projet sur le ",
      footerLink: "journal",
      footerAfter: ".",
    },
    contribute: {
      metaTitle: "Contribuer",
      metaDescription: "Comment contribuer à Pyrrhic Stars. Tout le projet est open source.",
      eyebrow: "Open source",
      heading: "Comment contribuer.",
      intro: "Tout le monde est bienvenu pour contribuer. L'ensemble est open source, sous licence AGPL. Je viens du développement backend, donc les domaines où j'ai le plus besoin d'aide sont sans doute évidents pour quiconque a essayé la version jouable.",
      needHeading: "Là où j'ai le plus besoin d'aide",
      needs: [
        { name: "Art et animation.", desc: "Graphismes, bons matériaux PBR, et surtout rigging, maillages et animation. C'est de loin mon point le plus faible." },
        { name: "Le monde.", desc: "Plus de nuance : PNJ, décor, détails." },
        { name: "Son et musique.", desc: "Extrêmement bienvenus." },
        { name: "Code.", desc: "Débogage, avancement des fonctionnalités, refactorisation." },
        { name: "Retours.", desc: "Ce qui fait du bien, ce qui tombe à plat, ce qui prête à confusion. Honnêtement, la chose la plus utile à ce stade." },
      ],
      startHeading: "Pour commencer",
      startBefore: "Le code se trouve ",
      startLink: "sur le dépôt public",
      startAfter: ". Ouvrez une issue ou une pull request.",
      startNoRepo: "Le lien vers le dépôt public arrive bientôt.",
      touchHeading: "Prendre contact",
      touchBefore: "Dites bonjour ",
      touchLink: "sur Discord",
      touchAfter: ".",
      touchNoDiscord: "Un Discord arrive bientôt.",
    },
  },
};

export function getContent(lang: Lang): PageContent {
  return content[lang] ?? content[defaultLang];
}
