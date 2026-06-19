// Open Graph card catalogue. Single source of truth for the per-route social
// images generated at build time by src/pages/og/[...route].png.ts, and for the
// image URL emitted by src/components/BaseHead.astro. Both sides derive the card
// id from the page pathname via ogId(), so the card a page references always
// exists and never drifts from the page's own <title>/description.

import { getCollection } from "astro:content";
import { SITE_TITLE } from "../consts";
import { getContent } from "../i18n/pages";
import { ui, languages, type Lang } from "../i18n/ui";

export interface OgCard {
  /** Route key, e.g. "index", "fr/about", "devlog/2026-06-02-hello-world". */
  id: string;
  title: string;
  description: string;
  lang: Lang;
}

/**
 * Map a page pathname to its card id. "/" -> "index"; locale prefixes and post
 * slugs are preserved verbatim ("/fr/about" -> "fr/about"). The result is the
 * key both the OG endpoint (getStaticPaths) and BaseHead use, so they stay in
 * lockstep without a shared lookup table.
 */
export function ogId(pathname: string): string {
  const trimmed = pathname.replace(/^\/+|\/+$/g, "");
  return trimmed === "" ? "index" : trimmed;
}

/** Build the catalogue of every card to render: each page in en + fr, plus the
 *  English-authored devlog posts. */
export async function ogCards(): Promise<OgCard[]> {
  const cards: OgCard[] = [];

  for (const lang of Object.keys(languages) as Lang[]) {
    const c = getContent(lang);
    const t = ui[lang];
    const prefix = lang === "en" ? "" : `${lang}/`;

    cards.push({ id: `${prefix}index`, title: SITE_TITLE, description: t["meta.home.description"], lang });
    cards.push({ id: `${prefix}about`, title: c.about.metaTitle, description: c.about.metaDescription, lang });
    cards.push({ id: `${prefix}roadmap`, title: c.roadmap.metaTitle, description: c.roadmap.metaDescription, lang });
    cards.push({ id: `${prefix}contribute`, title: c.contribute.metaTitle, description: c.contribute.metaDescription, lang });
    cards.push({ id: `${prefix}devlog`, title: t["devlogIndex.metaTitle"], description: t["devlogIndex.metaDescription"], lang });
  }

  // Devlog posts are authored in English only (frAlternate={false}), so each gets
  // a single English card keyed by its slug.
  const posts = await getCollection("devlog", ({ data }) => !data.draft);
  for (const post of posts) {
    cards.push({
      id: `devlog/${post.id}`,
      title: post.data.title,
      description: post.data.description,
      lang: "en",
    });
  }

  return cards;
}
