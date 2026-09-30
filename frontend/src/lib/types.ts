export interface Category {
  id: number
  name: string
  position: number
}

export interface Feed {
  id: number
  category_id: number | null
  title: string
  feed_url: string
  site_url: string
  description: string
  last_fetched_at: string | null
  last_error: string
  /** Stable failure code (e.g. "fetch_timeout"); last_error is the English detail. */
  last_error_code: string
  error_count: number
  created_at: string
}

export interface Entry {
  id: number
  feed_id: number
  url: string
  title: string
  author: string
  summary: string
  content?: string
  image_url: string
  published_at: string
  is_read: boolean
  is_starred: boolean
  /** Why a recommended entry was picked; only set on picks. */
  reason?: PickReason
}

/** Stable reason codes from GET /entries/recommended. */
export type PickReason = "favorite_source" | "rare_source" | "fresh"

export interface EntryPage {
  entries: Entry[]
  next_cursor: string | null
}

export interface Counters {
  unread: number
  starred: number
  feeds: Record<string, number>
  refreshing: boolean
}

export type FilterAction = "mark_read" | "skip"

/** A keyword rule applied to new articles; feed_id null means all feeds. */
export interface Filter {
  id: number
  feed_id: number | null
  keywords: string[]
  /** Also look in the article text, not just the title. */
  match_content: boolean
  /** Apply the action when none of the keywords appear. */
  invert: boolean
  action: FilterAction
  created_at: string
}

export type FilterInput = Omit<Filter, "id" | "created_at">

export interface Settings {
  refresh_interval_minutes: number
  retention_days: number
}

export interface AuthStatus {
  /** "development" disables sign-in; production is the default. */
  mode: "production" | "development"
  /** No admin account exists yet: show the setup page. */
  setup_required: boolean
  authenticated: boolean
  username?: string
}

export interface Account {
  username: string
  created_at: string
  updated_at: string
}

export interface AccountInput {
  current_password: string
  username: string
  /** Empty keeps the current password. */
  new_password?: string
}

/** A signed-in browser. */
export interface Session {
  id: number
  user_agent: string
  ip: string
  created_at: string
  last_seen_at: string
  expires_at: string
  /** The session making this request. */
  current: boolean
}

/** Which slice of entries the list is showing. */
export interface EntryFilter {
  feedId?: number
  categoryId?: number
  starred?: boolean
  unread?: boolean
  q?: string
}
