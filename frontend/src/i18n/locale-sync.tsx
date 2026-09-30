import { useEffect, useState } from "react"

import { usePrefs } from "@/lib/prefs"

import i18n, { resolveLocale } from "."

/** Applies the language preference to i18next and <html lang>. */
export function LocaleSync() {
  const [prefs] = usePrefs()
  const [languagesVersion, setLanguagesVersion] = useState(0)

  useEffect(() => {
    const onChange = () => setLanguagesVersion((n) => n + 1)
    window.addEventListener("languagechange", onChange)
    return () => window.removeEventListener("languagechange", onChange)
  }, [])

  useEffect(() => {
    const locale = resolveLocale(prefs.language)
    if (i18n.language !== locale) void i18n.changeLanguage(locale)
    document.documentElement.lang = locale
  }, [prefs.language, languagesVersion])

  return null
}
