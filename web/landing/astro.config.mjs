// @ts-check

import { defineConfig } from "astro/config";

import mdx from "@astrojs/mdx";
import sitemap from "@astrojs/sitemap";
import node from "@astrojs/node";
import icon from "astro-icon";

import tailwindcss from "@tailwindcss/vite";

// Canonical origin used by sitemap, RSS, and OG tags. Override at build time
// with PUBLIC_SITE_URL=https://your.domain pnpm run build.
const SITE = process.env.PUBLIC_SITE_URL || "http://localhost:4321";

// https://astro.build/config
export default defineConfig({
  site: SITE,

  // Server output with the Node adapter. Every route currently sets
  // `export const prerender = true`, so the server only serves static pages
  // today; the adapter is kept so a dynamic endpoint can be added back without
  // re-architecting the build.
  output: "server",
  adapter: node({ mode: "standalone" }),

  // English at the root (`/`), French under `/fr/`. The default locale is not
  // prefixed so existing URLs are untouched. The site is prerendered/static, so
  // browser-locale detection happens client-side (see LangRedirect.astro), not
  // via Accept-Language middleware.
  i18n: {
    locales: ["en", "fr"],
    defaultLocale: "en",
    routing: { prefixDefaultLocale: false },
  },

  integrations: [
    mdx(),
    sitemap({
      i18n: {
        defaultLocale: "en",
        locales: { en: "en", fr: "fr" },
      },
    }),
    icon(),
  ],

  vite: {
    plugins: [tailwindcss()],
  },
});
