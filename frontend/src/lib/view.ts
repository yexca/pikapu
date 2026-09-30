import { useMemo } from "react"
import { useLocation } from "react-router"

import type { EntryFilter } from "./types"

/** The entry collection selected in the sidebar, encoded in the URL path. */
export type View =
  | { kind: "all" }
  | { kind: "starred" }
  | { kind: "feed"; id: number }
  | { kind: "category"; id: number }

export function parseView(pathname: string): View {
  if (pathname.startsWith("/starred")) return { kind: "starred" }
  const feed = pathname.match(/^\/feeds\/(\d+)/)
  if (feed) return { kind: "feed", id: Number(feed[1]) }
  const category = pathname.match(/^\/categories\/(\d+)/)
  if (category) return { kind: "category", id: Number(category[1]) }
  return { kind: "all" }
}

export function viewPath(view: View): string {
  switch (view.kind) {
    case "starred":
      return "/starred"
    case "feed":
      return `/feeds/${view.id}`
    case "category":
      return `/categories/${view.id}`
    default:
      return "/"
  }
}

export function sameView(a: View, b: View): boolean {
  return viewPath(a) === viewPath(b)
}

export function useView(): View {
  const { pathname } = useLocation()
  return useMemo(() => parseView(pathname), [pathname])
}

export function viewFilter(
  view: View,
  unreadOnly: boolean,
  q: string
): EntryFilter {
  const searching = q.trim() !== ""
  const f: EntryFilter = {}
  if (view.kind === "feed") f.feedId = view.id
  if (view.kind === "category") f.categoryId = view.id
  if (view.kind === "starred") f.starred = true
  else if (unreadOnly && !searching) f.unread = true
  if (searching) f.q = q.trim()
  return f
}
