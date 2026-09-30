import { useCallback, useSyncExternalStore } from "react"

import type { LocalePreference } from "@/i18n"

/** Per-browser reading preferences, kept in localStorage. */
export interface Prefs {
  language: LocalePreference
  unreadOnly: boolean
  autoMarkRead: boolean
  fontSize: "sm" | "base" | "lg"
  /** Classic three panes, or the hub: picks first, updates grouped by source. */
  layout: "classic" | "hub"
  /** Category ids whose feed list is collapsed in the sidebar. */
  collapsed: number[]
}

const KEY = "pikapu-prefs"

const defaults: Prefs = {
  language: "auto",
  unreadOnly: true,
  autoMarkRead: true,
  fontSize: "base",
  layout: "classic",
  collapsed: [],
}

function load(): Prefs {
  try {
    const raw = localStorage.getItem(KEY)
    if (raw) return { ...defaults, ...JSON.parse(raw) }
  } catch {
    // Unavailable or corrupt storage: use defaults.
  }
  return defaults
}

let state = load()
const listeners = new Set<() => void>()

function subscribe(fn: () => void) {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function getPrefs(): Prefs {
  return state
}

export function setPrefs(patch: Partial<Prefs>) {
  state = { ...state, ...patch }
  try {
    localStorage.setItem(KEY, JSON.stringify(state))
  } catch {
    // ignore
  }
  listeners.forEach((fn) => fn())
}

export function usePrefs(): [Prefs, (patch: Partial<Prefs>) => void] {
  const prefs = useSyncExternalStore(subscribe, () => state)
  const update = useCallback((patch: Partial<Prefs>) => setPrefs(patch), [])
  return [prefs, update]
}
