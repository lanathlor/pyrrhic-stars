// Short UI strings, keyed by locale. Longer prose blocks live in ./pages.ts and
// the data modules (src/lib/classes.ts, src/lib/roadmap.ts).
//
// Keys are flat and dotted so `t("nav.home")` reads naturally. Any key missing
// from a non-default locale falls back to `defaultLang` (see ./utils.ts).
//
// Constraints (web/landing/CLAUDE.md): never use the word "MMO"; the hero
// tagline is the fixed line from marketing.md and the French is a faithful
// translation of it, not a rewrite.

export const defaultLang = "en";

export const languages = {
  en: "English",
  fr: "Français",
} as const;

export type Lang = keyof typeof languages;

export const ui = {
  en: {
    "nav.home": "Home",
    "nav.about": "About",
    "nav.roadmap": "Roadmap",
    "nav.devlog": "Devlog",
    "nav.contribute": "Contribute",

    "header.github": "GitHub repository",
    "lang.switch": "Language",
    "lang.en": "EN",
    "lang.fr": "FR",

    "hero.badge": "Phase 2 · More bosses",
    "hero.openSource": "Open source",
    "hero.tagline":
      "A co-op action game where every class plays a different genre.",
    "hero.description":
      "A sci-fi co-op dungeon crawler where the Gunner plays an FPS, the Vanguard plays a Souls-like, and the Blade Dancer chains combos, all in the same fight, against the same boss.",
    "hero.weeklyDevlog": "New devlog every week.",

    "features.eyebrow": "What makes it different",
    "features.heading": "Four pillars.",

    "classes.eyebrow": "The roster",
    "classes.heading": "Six classes. One dungeon.",
    "classes.playable": "Playable",
    "classes.planned": "Planned",

    "roadmap.eyebrow": "Where this is going",
    "roadmap.heading": "Roadmap.",
    "roadmap.intro":
      "Built in phases. Each phase ends with something concrete: a clip, a playable build, a Steam page. Before the next one starts.",
    "roadmap.readFull": "Read the full roadmap",
    "roadmap.status.now": "Now",
    "roadmap.status.done": "Done",
    "roadmap.status.next": "Next",

    "devlogPreview.eyebrow": "Build in public",
    "devlogPreview.heading": "From the devlog.",
    "devlogPreview.viewAll": "View all",

    "devlogCard.read": "Read",

    "join.eyebrow": "Join the community",
    "join.heading": "Follow along on Discord.",
    "join.body":
      "I post progress there as it happens. Come hang out, or tell me what is not working.",

    "cta.joinDiscord": "Join Discord",
    "cta.discordSoon": "Discord: Soon",
    "cta.downloadLinux": "Download (Linux)",
    "cta.linuxSoon": "Linux: Soon",
    "cta.downloadWindows": "Download (Windows)",
    "cta.windowsSoon": "Windows: Soon",
    "cta.wishlistSteam": "Wishlist on Steam",

    "footer.builtInPublic": "Built in public.",
    "footer.devlog": "Devlog",
    "footer.rss": "RSS",
    "footer.github": "GitHub",
    "footer.discord": "Discord",

    "devlogPost.back": "Back to devlog",
    "devlogPost.updated": "updated",

    "devlogIndex.eyebrow": "Build in public",
    "devlogIndex.heading": "Devlog.",
    "devlogIndex.intro":
      "One concrete thing working each week. New systems, new footage, real progress.",
    "devlogIndex.empty": "No posts yet. Check back soon.",
    "devlogIndex.metaTitle": "Devlog",
    "devlogIndex.metaDescription":
      "Weekly build-in-public updates from the Pyrrhic Stars dev team.",

    "meta.home.description":
      "Pyrrhic Stars: a co-op action game where every class plays a different genre. FPS, Souls-like, tactical channeling, deployables, blade combos, aura positioning. Build in public, weekly devlog.",
  },

  fr: {
    "nav.home": "Accueil",
    "nav.about": "À propos",
    "nav.roadmap": "Feuille de route",
    "nav.devlog": "Journal",
    "nav.contribute": "Contribuer",

    "header.github": "Dépôt GitHub",
    "lang.switch": "Langue",
    "lang.en": "EN",
    "lang.fr": "FR",

    "hero.badge": "Phase 2 · Plus de boss",
    "hero.openSource": "Open source",
    "hero.tagline":
      "Un jeu d'action coopératif où chaque classe joue à un genre différent.",
    "hero.description":
      "Un dungeon crawler coopératif de science-fiction où le Tireur joue à un FPS, l'Avant-garde à un Souls-like et le Danseur de lames enchaîne les combos, le tout dans le même combat, contre le même boss.",
    "hero.weeklyDevlog": "Un nouveau journal chaque semaine.",

    "features.eyebrow": "Ce qui le rend différent",
    "features.heading": "Quatre piliers.",

    "classes.eyebrow": "Le roster",
    "classes.heading": "Six classes. Un donjon.",
    "classes.playable": "Jouable",
    "classes.planned": "Prévu",

    "roadmap.eyebrow": "Où tout cela mène",
    "roadmap.heading": "Feuille de route.",
    "roadmap.intro":
      "Construit par phases. Chaque phase se termine par quelque chose de concret : un clip, une version jouable, une page Steam. Avant que la suivante ne commence.",
    "roadmap.readFull": "Lire la feuille de route complète",
    "roadmap.status.now": "En cours",
    "roadmap.status.done": "Fait",
    "roadmap.status.next": "Ensuite",

    "devlogPreview.eyebrow": "Développement ouvert",
    "devlogPreview.heading": "Depuis le journal.",
    "devlogPreview.viewAll": "Tout voir",

    "devlogCard.read": "Lire",

    "join.eyebrow": "Rejoindre la communauté",
    "join.heading": "Suivez le projet sur Discord.",
    "join.body":
      "J'y partage les avancées au fil de l'eau. Venez discuter, ou dites-moi ce qui ne marche pas.",

    "cta.joinDiscord": "Rejoindre Discord",
    "cta.discordSoon": "Discord : bientôt",
    "cta.downloadLinux": "Télécharger (Linux)",
    "cta.linuxSoon": "Linux : bientôt",
    "cta.downloadWindows": "Télécharger (Windows)",
    "cta.windowsSoon": "Windows : bientôt",
    "cta.wishlistSteam": "Ajouter à la liste de souhaits Steam",

    "footer.builtInPublic": "Développé en public.",
    "footer.devlog": "Journal",
    "footer.rss": "RSS",
    "footer.github": "GitHub",
    "footer.discord": "Discord",

    "devlogPost.back": "Retour au journal",
    "devlogPost.updated": "mis à jour",

    "devlogIndex.eyebrow": "Développement ouvert",
    "devlogIndex.heading": "Journal.",
    "devlogIndex.intro":
      "Une chose concrète qui fonctionne chaque semaine. Nouveaux systèmes, nouvelles images, de vrais progrès.",
    "devlogIndex.empty": "Aucun article pour l'instant. Revenez bientôt.",
    "devlogIndex.metaTitle": "Journal",
    "devlogIndex.metaDescription":
      "Mises à jour hebdomadaires du développement ouvert de Pyrrhic Stars.",

    "meta.home.description":
      "Pyrrhic Stars : un jeu d'action coopératif où chaque classe joue à un genre différent. FPS, Souls-like, incantation tactique, déploiements, combos de lames, positionnement d'auras. Développé en public, journal hebdomadaire.",
  },
} as const;
