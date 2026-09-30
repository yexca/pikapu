/** Mirrors the server's password rule (8–128 characters). */
export const MIN_PASSWORD_LENGTH = 8

const browsers: [RegExp, string][] = [
  [/\bEdg(e|A|iOS)?\//, "Edge"],
  [/\b(OPR|Opera)\//, "Opera"],
  [/\bFirefox\/|\bFxiOS\//, "Firefox"],
  [/\bSamsungBrowser\//, "Samsung Internet"],
  [/\bChrome\/|\bCriOS\//, "Chrome"],
  [/\bSafari\//, "Safari"],
]

const systems: [RegExp, string][] = [
  [/\biPhone\b/, "iPhone"],
  [/\biPad\b/, "iPad"],
  [/\bAndroid\b/, "Android"],
  [/\bWindows\b/, "Windows"],
  [/\bCrOS\b/, "ChromeOS"],
  [/\bMac OS X\b|\bMacintosh\b/, "macOS"],
  [/\bLinux\b/, "Linux"],
]

/** A short "Browser · OS" label for a session, or "" when unrecognized. */
export function describeUserAgent(ua: string): string {
  const browser = browsers.find(([re]) => re.test(ua))?.[1]
  const os = systems.find(([re]) => re.test(ua))?.[1]
  return [browser, os].filter(Boolean).join(" · ")
}
