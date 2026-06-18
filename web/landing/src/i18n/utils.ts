// i18n helpers. Locale is derived from the URL (English at `/`, French under
// `/fr/`), so components can resolve their own language without prop drilling:
//
//   const lang = getLang(Astro);
//   const t = useTranslations(lang);
//
// Path helpers build locale-correct internal links and the language-switch /
// hreflang targets. They intentionally avoid astro:i18n's getRelativeLocaleUrl
// so trailing-slash behaviour stays under our control.

import { ui, defaultLang, languages, type Lang } from "./ui";

export { defaultLang, languages };
export type { Lang };

const PREFIX: Record<Lang, string> = { en: "", fr: "/fr" };

/** True for keys that exist in the default locale dictionary. */
type UIKey = keyof (typeof ui)[typeof defaultLang];

export function isLang(value: unknown): value is Lang {
  return value === "en" || value === "fr";
}

/** Resolve the active locale from Astro.currentLocale, falling back to default. */
export function getLang(astro: { currentLocale?: string }): Lang {
  return isLang(astro.currentLocale) ? astro.currentLocale : defaultLang;
}

/** Returns a `t(key)` that looks up `lang`, falling back to the default locale. */
export function useTranslations(lang: Lang) {
  return function t(key: UIKey): string {
    return ui[lang][key] ?? ui[defaultLang][key];
  };
}

/** Strip any locale prefix, returning the canonical (English) path. */
export function neutralPath(pathname: string): string {
  if (pathname === "/fr" || pathname === "/fr/") return "/";
  if (pathname.startsWith("/fr/")) return pathname.slice(3);
  return pathname;
}

/** Build the URL for `path` (a neutral path like "/about") in `lang`. */
export function localePath(lang: Lang, path: string): string {
  const clean = path.startsWith("/") ? path : `/${path}`;
  if (lang === defaultLang) return clean;
  return clean === "/" ? `${PREFIX[lang]}/` : `${PREFIX[lang]}${clean}`;
}

/** Map the current pathname to its counterpart in `target` (switcher, hreflang). */
export function switchLocalePath(pathname: string, target: Lang): string {
  return localePath(target, neutralPath(pathname));
}
