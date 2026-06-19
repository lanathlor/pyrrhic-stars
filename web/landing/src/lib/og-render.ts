// Renders an OG card to a 1200x630 PNG. satori turns a flexbox tree into SVG
// (text becomes vector paths using the embedded fonts, so no system fonts are
// needed downstream); sharp rasterises that SVG to PNG. Build-time only - called
// from the prerendered endpoint in src/pages/og/[...route].png.ts.

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import satori from "satori";
import sharp from "sharp";
import { getContent } from "../i18n/pages";
import type { Lang } from "../i18n/ui";
import type { OgCard } from "./og";

// Resolve fonts from the package root rather than import.meta.url: this module is
// bundled into dist/server/.prerender at build, which breaks URL-relative paths.
// These cards are prerendered, so this only runs during `astro build`, where the
// working directory is the landing package.
const fontDir = resolve(process.cwd(), "src/assets/og");
const interRegular = readFileSync(resolve(fontDir, "Inter-Regular.otf"));
const interSemiBold = readFileSync(resolve(fontDir, "Inter-SemiBold.otf"));

// Brand palette, mirrored from public/favicon.svg and docs/design/ui-language.md.
const BG_TOP = "#0e1318";
const BG_BOTTOM = "#171c22";
const ACCENT = "#6e90b5";
const TEXT_PRIMARY = "#e6ebef";
const TEXT_SECONDARY = "#90a0ad";
const DIVIDER = "#2a333c";

// satori element helper (this file is plain .ts, no JSX).
type Node = string | { type: string; props: Record<string, unknown> };
function h(type: string, style: Record<string, unknown>, children?: Node | Node[]): Node {
  return { type, props: { style, children } };
}

/** Keep the description to a single readable block so it never overflows the card. */
function clamp(text: string, max = 150): string {
  if (text.length <= max) return text;
  return `${text.slice(0, max - 1).trimEnd()}…`;
}

function tagsFor(lang: Lang): string {
  const tags = getContent(lang).features[0]?.tags ?? [];
  return tags.join("  ·  ");
}

function template(card: OgCard): Node {
  return h(
    "div",
    {
      width: 1200,
      height: 630,
      display: "flex",
      flexDirection: "column",
      justifyContent: "space-between",
      padding: "72px 80px",
      backgroundColor: BG_TOP,
      backgroundImage: `linear-gradient(135deg, ${BG_TOP}, ${BG_BOTTOM})`,
      borderLeft: `10px solid ${ACCENT}`,
      fontFamily: "Inter",
    },
    [
      // Eyebrow wordmark.
      h(
        "div",
        {
          display: "flex",
          fontSize: 26,
          fontWeight: 600,
          letterSpacing: 6,
          color: ACCENT,
        },
        "PYRRHIC STARS",
      ),
      // Title + description.
      h(
        "div",
        {
          display: "flex",
          flexDirection: "column",
          flexGrow: 1,
          justifyContent: "center",
        },
        [
          h(
            "div",
            {
              display: "flex",
              fontSize: 72,
              fontWeight: 600,
              lineHeight: 1.05,
              letterSpacing: -1.5,
              color: TEXT_PRIMARY,
            },
            card.title,
          ),
          h(
            "div",
            {
              display: "flex",
              marginTop: 28,
              maxWidth: 940,
              fontSize: 30,
              fontWeight: 400,
              lineHeight: 1.4,
              color: TEXT_SECONDARY,
            },
            clamp(card.description),
          ),
        ],
      ),
      // Divider + localized genre tag strip.
      h(
        "div",
        { display: "flex", flexDirection: "column" },
        [
          h("div", { display: "flex", height: 1, width: "100%", backgroundColor: DIVIDER, marginBottom: 24 }),
          h(
            "div",
            { display: "flex", fontSize: 22, fontWeight: 600, letterSpacing: 0.5, color: ACCENT },
            tagsFor(card.lang),
          ),
        ],
      ),
    ],
  );
}

export async function renderCard(card: OgCard): Promise<Buffer> {
  const svg = await satori(template(card) as Parameters<typeof satori>[0], {
    width: 1200,
    height: 630,
    fonts: [
      { name: "Inter", data: interRegular, weight: 400, style: "normal" },
      { name: "Inter", data: interSemiBold, weight: 600, style: "normal" },
    ],
  });
  return sharp(Buffer.from(svg)).png().toBuffer();
}
