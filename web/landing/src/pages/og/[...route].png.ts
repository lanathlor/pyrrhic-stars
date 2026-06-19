// Prerendered per-route Open Graph images. Emits one PNG per card id, e.g.
// /og/index.png, /og/fr/about.png, /og/devlog/<slug>.png. The catalogue and the
// pathname->id mapping live in src/lib/og.ts so BaseHead references match.

import type { APIRoute, GetStaticPaths } from "astro";
import { ogCards, type OgCard } from "../../lib/og";
import { renderCard } from "../../lib/og-render";

export const prerender = true;

export const getStaticPaths: GetStaticPaths = async () => {
  const cards = await ogCards();
  return cards.map((card) => ({ params: { route: card.id }, props: { card } }));
};

export const GET: APIRoute = async ({ props }) => {
  const png = await renderCard((props as { card: OgCard }).card);
  // Node Buffer isn't a DOM BodyInit; copy into an ArrayBuffer-backed Uint8Array.
  const body = Uint8Array.from(png);
  return new Response(body, {
    headers: {
      "Content-Type": "image/png",
      "Cache-Control": "public, max-age=31536000, immutable",
    },
  });
};
