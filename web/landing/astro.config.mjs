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

  // Server-rendered by default; individual pages opt into prerendering with
  // `export const prerender = true`. The only non-prerendered route is the
  // /api/subscribe POST handler.
  output: "server",
  adapter: node({ mode: "standalone" }),

  integrations: [mdx(), sitemap(), icon()],

  vite: {
    plugins: [tailwindcss()],
  },
});
