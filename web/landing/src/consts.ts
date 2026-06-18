// Site-wide constants. Keep in sync with docs/project/marketing.md.

export const SITE_TITLE = "Pyrrhic Stars";

// Discord invite. The final site is a static GitHub Pages build with no runtime
// env, so the link is committed here as the default. PUBLIC_DISCORD_URL still
// wins when set (e.g. preview builds pointing at a different server).
export const DISCORD_URL =
  import.meta.env.PUBLIC_DISCORD_URL || "https://discord.gg/UD5cChCGtd";

// Public source repository. Like DISCORD_URL, committed here as the default for
// the static build. PUBLIC_REPO_URL still wins when set.
export const REPO_URL =
  import.meta.env.PUBLIC_REPO_URL || "https://github.com/lanathlor/pyrrhic-stars";

// Sourced verbatim from docs/project/marketing.md:5. Do not rewrite.
export const SITE_TAGLINE =
  "A co-op action game where every class plays a different genre.";

export const SITE_DESCRIPTION =
  "Pyrrhic Stars: a co-op action game where every class plays a different genre. FPS, Souls-like, tactical channeling, deployables, blade combos, aura positioning. Build in public, weekly devlog.";

// Navigation. `href` is a locale-neutral path (localized at render time via
// localePath); `key` resolves the label through the i18n dictionary.
export interface NavItem {
  href: string;
  key:
    | "nav.home"
    | "nav.about"
    | "nav.roadmap"
    | "nav.devlog"
    | "nav.contribute";
}

export const NAV: NavItem[] = [
  { href: "/", key: "nav.home" },
  { href: "/about", key: "nav.about" },
  { href: "/roadmap", key: "nav.roadmap" },
  { href: "/devlog", key: "nav.devlog" },
  { href: "/contribute", key: "nav.contribute" },
];
