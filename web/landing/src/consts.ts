// Site-wide constants. Keep in sync with docs/project/marketing.md.

export const SITE_TITLE = "Pyrrhic Stars";

// Sourced verbatim from docs/project/marketing.md:5. Do not rewrite.
export const SITE_TAGLINE =
  "A co-op action game where every class plays a different genre.";

export const SITE_DESCRIPTION =
  "Pyrrhic Stars: a co-op action game where every class plays a different genre. FPS, Souls-like, tactical channeling, deployables, blade combos, aura positioning. Build in public, weekly devlog.";

export interface NavItem {
  href: string;
  label: string;
}

export const NAV: NavItem[] = [
  { href: "/", label: "Home" },
  { href: "/roadmap", label: "Roadmap" },
  { href: "/devlog", label: "Devlog" },
];
