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
}

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

export interface Settings {
  refresh_interval_minutes: number
  retention_days: number
}

export interface AuthStatus {
  auth_required: boolean
  authenticated: boolean
}

/** Which slice of entries the list is showing. */
export interface EntryFilter {
  feedId?: number
  categoryId?: number
  starred?: boolean
  unread?: boolean
  q?: string
}
