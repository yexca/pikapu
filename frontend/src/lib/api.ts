import type {
  AuthStatus,
  Category,
  Counters,
  Entry,
  EntryFilter,
  EntryPage,
  Feed,
  Settings,
} from "./types"

/** An API failure: `message` is the server's English text, `code` is stable. */
export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

let onUnauthorized: (() => void) | undefined

/** Registers a callback fired whenever the server answers 401. */
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const init: RequestInit = { method, credentials: "same-origin", headers: {} }
  if (body instanceof FormData) {
    init.body = body
  } else if (body !== undefined) {
    init.body = JSON.stringify(body)
    init.headers = { "Content-Type": "application/json" }
  }

  let res: Response
  try {
    res = await fetch(`/api${path}`, init)
  } catch {
    throw new ApiError(0, "network_unreachable", "Cannot reach the server")
  }

  if (res.status === 401 && !path.startsWith("/auth/")) {
    onUnauthorized?.()
  }
  if (!res.ok) {
    let message = `Request failed (${res.status})`
    let code = ""
    try {
      const data = await res.json()
      if (data?.error) message = data.error
      if (data?.code) code = data.code
    } catch {
      // Non-JSON error body; keep the generic message.
    }
    throw new ApiError(res.status, code, message)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

function filterParams(f: EntryFilter): Record<string, string> {
  const p: Record<string, string> = {}
  if (f.feedId) p.feed_id = String(f.feedId)
  if (f.categoryId) p.category_id = String(f.categoryId)
  if (f.starred) p.starred = "1"
  if (f.unread) p.unread = "1"
  if (f.q?.trim()) p.q = f.q.trim()
  return p
}

export const api = {
  authStatus: () => request<AuthStatus>("GET", "/auth/status"),
  login: (password: string) =>
    request<{ authenticated: boolean }>("POST", "/auth/login", { password }),
  logout: () => request<void>("POST", "/auth/logout"),

  categories: () => request<Category[]>("GET", "/categories"),
  createCategory: (name: string) =>
    request<Category>("POST", "/categories", { name }),
  renameCategory: (id: number, name: string) =>
    request<Category>("PUT", `/categories/${id}`, { name }),
  deleteCategory: (id: number) => request<void>("DELETE", `/categories/${id}`),

  feeds: () => request<Feed[]>("GET", "/feeds"),
  addFeed: (input: {
    url: string
    category_id?: number | null
    category_name?: string
  }) => request<Feed>("POST", "/feeds", input),
  updateFeed: (
    id: number,
    input: {
      title: string
      feed_url: string
      category_id: number | null
      category_name?: string
    }
  ) => request<Feed>("PUT", `/feeds/${id}`, input),
  deleteFeed: (id: number) => request<void>("DELETE", `/feeds/${id}`),
  refreshFeed: (id: number) => request<Feed>("POST", `/feeds/${id}/refresh`),
  refreshAll: () => request<{ started: boolean }>("POST", "/feeds/refresh"),

  entries: (filter: EntryFilter, cursor?: string | null) => {
    const params = new URLSearchParams(filterParams(filter))
    if (cursor) params.set("cursor", cursor)
    return request<EntryPage>("GET", `/entries?${params}`)
  },
  entry: (id: number) => request<Entry>("GET", `/entries/${id}`),
  updateEntry: (
    id: number,
    patch: { is_read?: boolean; is_starred?: boolean }
  ) => request<void>("PATCH", `/entries/${id}`, patch),
  markAllRead: (filter: EntryFilter) =>
    request<{ updated: number }>("POST", "/entries/mark-all-read", {
      feed_id: filter.feedId ?? 0,
      category_id: filter.categoryId ?? 0,
      starred: !!filter.starred,
      q: filter.q?.trim() ?? "",
    }),

  counters: () => request<Counters>("GET", "/counters"),
  settings: () => request<Settings>("GET", "/settings"),
  saveSettings: (s: Settings) => request<Settings>("PUT", "/settings", s),
  importOpml: (file: File) => {
    const form = new FormData()
    form.append("file", file)
    return request<{ added: number; skipped: number }>("POST", "/opml", form)
  },
}

export const feedIconUrl = (id: number) => `/api/feeds/${id}/icon`
export const opmlExportUrl = "/api/opml"
