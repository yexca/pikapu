import i18n from "i18next"
import { initReactI18next } from "react-i18next"

import { getPrefs } from "@/lib/prefs"

import en from "./locales/en"
import zhHans from "./locales/zh-Hans"

export type Locale = "en" | "zh-Hans"
export type LocalePreference = "auto" | Locale

/** Selectable languages, labelled in their own language. */
export const LOCALES: readonly { value: Locale; label: string }[] = [
  { value: "en", label: "English" },
  { value: "zh-Hans", label: "简体中文" },
]

function systemLanguages(): string[] {
  if (typeof navigator === "undefined") return []
  return [...(navigator.languages ?? []), navigator.language].filter(Boolean)
}

/**
 * Picks the UI locale. "auto" takes the first supported language from the
 * browser's preference list; anything unsupported (including Traditional
 * Chinese) falls back to English.
 */
export function resolveLocale(
  preference: LocalePreference,
  languages: readonly string[] = systemLanguages()
): Locale {
  if (preference !== "auto") return preference
  for (const language of languages) {
    const tag = language.toLowerCase().replace(/_/g, "-")
    if (tag === "en" || tag.startsWith("en-")) return "en"
    if (
      tag === "zh" ||
      tag.startsWith("zh-hans") ||
      tag === "zh-cn" ||
      tag === "zh-sg" ||
      tag.startsWith("zh-cn-") ||
      tag.startsWith("zh-sg-")
    ) {
      return "zh-Hans"
    }
  }
  return "en"
}

/** BCP 47 tag for Intl formatters. */
export function intlLocale(locale: string = i18n.language): string {
  return locale === "zh-Hans" ? "zh-CN" : "en-US"
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    "zh-Hans": { translation: zhHans },
  },
  lng: resolveLocale(getPrefs().language),
  fallbackLng: "en",
  supportedLngs: ["en", "zh-Hans"],
  load: "currentOnly",
  interpolation: { escapeValue: false },
  returnNull: false,
  returnEmptyString: false,
  initAsync: false,
})

export default i18n
