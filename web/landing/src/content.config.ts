import { defineCollection } from "astro:content";
import { glob } from "astro/loaders";
import { z } from "astro/zod";

// Devlog collection. Mirrors the schema from
// /home/lanath/Work/lanath/cv/site/src/content.config.ts so authoring habits
// transfer between projects.
const devlog = defineCollection({
  loader: glob({ base: "./src/content/devlog", pattern: "**/*.{md,mdx}" }),
  schema: ({ image }) =>
    z.object({
      title: z.string(),
      description: z.string(),
      pubDate: z.coerce.date(),
      updatedDate: z.coerce.date().optional(),
      heroImage: z.optional(image()),
      draft: z.boolean().optional(),
    }),
});

export const collections = { devlog };
