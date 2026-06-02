/// <reference path="../.astro/types.d.ts" />

interface ImportMetaEnv {
  readonly PUBLIC_SITE_URL?: string;
  readonly PUBLIC_DISCORD_URL?: string;
  readonly PUBLIC_DOWNLOAD_LINUX?: string;
  readonly PUBLIC_DOWNLOAD_WINDOWS?: string;
  readonly PUBLIC_STEAM_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
