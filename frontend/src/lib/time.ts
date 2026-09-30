import i18n, { intlLocale } from "@/i18n"

// Formatter construction is the expensive part, and lists format many dates.
const relative = new Map<string, Intl.RelativeTimeFormat>()
const dates = new Map<string, Intl.DateTimeFormat>()

function relativeFormat(locale: string) {
  let f = relative.get(locale)
  if (!f) {
    f = new Intl.RelativeTimeFormat(locale, { numeric: "auto" })
    relative.set(locale, f)
  }
  return f
}

function dateFormat(locale: string, options: Intl.DateTimeFormatOptions) {
  const key = `${locale}|${JSON.stringify(options)}`
  let f = dates.get(key)
  if (!f) {
    f = new Intl.DateTimeFormat(locale, options)
    dates.set(key, f)
  }
  return f
}

/** Short relative time ("5 minutes ago"), or a date beyond a week. */
export function relativeTime(iso: string | null | undefined): string {
  if (!iso) return ""
  const locale = intlLocale()
  const date = new Date(iso)
  const diff = (date.getTime() - Date.now()) / 1000
  const abs = Math.abs(diff)
  if (abs < 60) return i18n.t("time.justNow")
  const rtf = relativeFormat(locale)
  if (abs < 3600) return rtf.format(Math.round(diff / 60), "minute")
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), "hour")
  if (abs < 7 * 86400) return rtf.format(Math.round(diff / 86400), "day")
  const sameYear = date.getFullYear() === new Date().getFullYear()
  return dateFormat(
    locale,
    sameYear
      ? { month: "short", day: "numeric" }
      : { year: "numeric", month: "short", day: "numeric" }
  ).format(date)
}

export function fullTime(iso: string): string {
  return dateFormat(intlLocale(), {
    year: "numeric",
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso))
}
