import {
  type InfiniteData,
  type QueryClient,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { toast } from "sonner"

import i18n from "@/i18n"
import { errorMessage, feedErrorMessage } from "@/i18n/errors"

import { api } from "./api"
import type {
  AccountInput,
  AuthStatus,
  Counters,
  Entry,
  EntryFilter,
  EntryPage,
  Feed,
  Filter,
  FilterInput,
  Settings,
} from "./types"

export const keys = {
  auth: ["auth"] as const,
  account: ["account"] as const,
  sessions: ["account", "sessions"] as const,
  categories: ["categories"] as const,
  feeds: ["feeds"] as const,
  counters: ["counters"] as const,
  settings: ["settings"] as const,
  filters: ["filters"] as const,
  entryLists: ["entries"] as const,
  entries: (f: EntryFilter) => ["entries", f] as const,
  // Under the "entries" prefix and in the same shape as a list, so read and
  // star updates, lookups, and invalidations cover picks too.
  picks: (scope: EntryFilter) => ["entries", "picks", scope] as const,
  entry: (id: number) => ["entry", id] as const,
}

export function useAuthStatus() {
  return useQuery({
    queryKey: keys.auth,
    queryFn: api.authStatus,
    staleTime: Infinity,
    retry: 1,
  })
}

export function useCategories() {
  return useQuery({ queryKey: keys.categories, queryFn: api.categories })
}

export function useFeeds() {
  return useQuery({ queryKey: keys.feeds, queryFn: api.feeds })
}

export function useCounters() {
  return useQuery({
    queryKey: keys.counters,
    queryFn: api.counters,
    // Poll quickly while the server is refreshing feeds.
    refetchInterval: (q) => (q.state.data?.refreshing ? 1500 : 60_000),
    refetchOnWindowFocus: true,
  })
}

export function useSettings() {
  return useQuery({ queryKey: keys.settings, queryFn: api.settings })
}

export function useEntries(filter: EntryFilter) {
  return useInfiniteQuery({
    queryKey: keys.entries(filter),
    queryFn: ({ pageParam }) => api.entries(filter, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor,
  })
}

/** Canonical picks scope, so keys built from a view and a filter match. */
export function pickScope({ feedId, categoryId }: EntryFilter): EntryFilter {
  if (categoryId) return { categoryId }
  if (feedId) return { feedId }
  return {}
}

/** Recommended entries for a view; ranked once per load so they stay put. */
export function usePicks(filter: EntryFilter, enabled: boolean) {
  const scope = pickScope(filter)
  return useInfiniteQuery({
    queryKey: keys.picks(scope),
    queryFn: () => api.recommended(scope),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor,
    enabled,
  })
}

export function useEntry(id: number | null) {
  return useQuery({
    queryKey: keys.entry(id ?? 0),
    queryFn: () => api.entry(id!),
    enabled: id != null,
    staleTime: 5 * 60_000,
  })
}

export function prefetchEntry(qc: QueryClient, id: number) {
  return qc.prefetchQuery({
    queryKey: keys.entry(id),
    queryFn: () => api.entry(id),
    staleTime: 5 * 60_000,
  })
}

type EntryPatch = Partial<Pick<Entry, "is_read" | "is_starred">>

function patchEntryLists(
  qc: QueryClient,
  fn: (e: Entry) => Entry,
  queryKey: readonly unknown[] = keys.entryLists
) {
  qc.setQueriesData<InfiniteData<EntryPage>>(
    { queryKey },
    (data) =>
      data && {
        ...data,
        pages: data.pages.map((p) => ({ ...p, entries: p.entries.map(fn) })),
      }
  )
}

/** Returns the freshest cached copy of an entry, if any. */
export function findCachedEntry(
  qc: QueryClient,
  id: number
): Entry | undefined {
  const detail = qc.getQueryData<Entry>(keys.entry(id))
  if (detail) return detail
  for (const [, data] of qc.getQueriesData<InfiniteData<EntryPage>>({
    queryKey: keys.entryLists,
  })) {
    for (const page of data?.pages ?? []) {
      const e = page.entries.find((x) => x.id === id)
      if (e) return e
    }
  }
  return undefined
}

function applyEntryPatch(qc: QueryClient, id: number, patch: EntryPatch) {
  const current = findCachedEntry(qc, id)
  patchEntryLists(qc, (e) => (e.id === id ? { ...e, ...patch } : e))
  qc.setQueryData<Entry>(keys.entry(id), (e) => e && { ...e, ...patch })

  if (!current) return
  qc.setQueryData<Counters>(keys.counters, (c) => {
    if (!c) return c
    const next = { ...c, feeds: { ...c.feeds } }
    if (patch.is_read !== undefined && patch.is_read !== current.is_read) {
      const d = patch.is_read ? -1 : 1
      const fid = String(current.feed_id)
      next.feeds[fid] = Math.max(0, (next.feeds[fid] ?? 0) + d)
      next.unread = Math.max(0, next.unread + d)
    }
    if (
      patch.is_starred !== undefined &&
      patch.is_starred !== current.is_starred
    ) {
      next.starred = Math.max(0, next.starred + (patch.is_starred ? 1 : -1))
    }
    return next
  })
}

/** Toggles read/starred with an optimistic cache update. */
export function useUpdateEntry() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: number; patch: EntryPatch }) =>
      api.updateEntry(id, patch),
    onMutate: ({ id, patch }) => {
      const before = findCachedEntry(qc, id)
      applyEntryPatch(qc, id, patch)
      return { before }
    },
    onError: (err, { id }, ctx) => {
      if (ctx?.before) {
        const { is_read, is_starred } = ctx.before
        applyEntryPatch(qc, id, { is_read, is_starred })
      }
      qc.invalidateQueries({ queryKey: keys.counters })
      toast.error(errorMessage(err))
    },
  })
}

export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (filter: EntryFilter) => api.markAllRead(filter),
    onSuccess: (res, filter) => {
      // Keep the visible list in place, just dimmed; other lists refetch later.
      const read = (e: Entry) => ({ ...e, is_read: true })
      patchEntryLists(qc, read, keys.entries(filter))
      if (!filter.q && !filter.starred) {
        patchEntryLists(qc, read, keys.picks(pickScope(filter)))
      }
      qc.invalidateQueries({ queryKey: keys.entryLists, refetchType: "none" })
      qc.invalidateQueries({ queryKey: ["entry"], refetchType: "none" })
      qc.invalidateQueries({ queryKey: keys.counters })
      toast.success(
        res.updated > 0
          ? i18n.t("list.markedRead", { count: res.updated })
          : i18n.t("list.nothingToMark")
      )
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

// Set when the user explicitly asks for a refresh, so that the entry list is
// reloaded once the background refresh completes. Scheduled refreshes only
// update the counters to avoid shuffling the list while reading.
let userRequestedRefresh = false

export function consumeUserRefresh(): boolean {
  const v = userRequestedRefresh
  userRequestedRefresh = false
  return v
}

export function useRefreshAll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.refreshAll,
    onSuccess: () => {
      userRequestedRefresh = true
      qc.setQueryData<Counters>(
        keys.counters,
        (c) => c && { ...c, refreshing: true }
      )
      qc.invalidateQueries({ queryKey: keys.counters })
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useRefreshFeed() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.refreshFeed(id),
    onSuccess: (feed) => {
      qc.setQueryData<Feed[]>(keys.feeds, (list) =>
        list?.map((f) => (f.id === feed.id ? feed : f))
      )
      qc.invalidateQueries({ queryKey: keys.counters })
      qc.invalidateQueries({ queryKey: keys.entryLists })
      if (feed.last_error) {
        toast.error(
          i18n.t("feed.updateFailed", { reason: feedErrorMessage(feed) })
        )
      } else {
        toast.success(i18n.t("feed.refreshed", { title: feed.title }))
      }
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

function invalidateSubscriptions(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: keys.feeds })
  qc.invalidateQueries({ queryKey: keys.categories })
  qc.invalidateQueries({ queryKey: keys.counters })
}

export function useAddFeed() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.addFeed,
    onSuccess: () => {
      invalidateSubscriptions(qc)
      qc.invalidateQueries({ queryKey: keys.entryLists })
    },
  })
}

export function useUpdateFeed() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...input
    }: {
      id: number
      title: string
      feed_url: string
      category_id: number | null
      category_name?: string
    }) => api.updateFeed(id, input),
    onSuccess: () => {
      invalidateSubscriptions(qc)
      qc.invalidateQueries({ queryKey: keys.entryLists })
    },
  })
}

export function useDeleteFeed() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteFeed(id),
    onSuccess: () => {
      invalidateSubscriptions(qc)
      qc.invalidateQueries({ queryKey: keys.entryLists })
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useSaveCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id?: number; name: string }) =>
      id ? api.renameCategory(id, name) : api.createCategory(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.categories }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteCategory(id),
    onSuccess: () => invalidateSubscriptions(qc),
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useFilters() {
  return useQuery({ queryKey: keys.filters, queryFn: api.filters })
}

export function useSaveFilter() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: FilterInput & { id?: number }) =>
      id ? api.updateFilter(id, input) : api.createFilter(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.filters }),
  })
}

export function useDeleteFilter() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteFilter(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.filters }),
    onError: (err) => toast.error(errorMessage(err)),
  })
}

/**
 * Runs a just-saved filter over current unread articles, which may remove
 * some. The toast lives here because the form unmounts before this settles.
 */
export function useApplyFilter() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (filter: Filter) => api.applyFilter(filter.id),
    onSuccess: ({ updated }, filter) => {
      if (updated === 0) {
        toast.success(i18n.t("filters.saved"))
        return
      }
      qc.invalidateQueries({ queryKey: keys.entryLists })
      qc.invalidateQueries({ queryKey: ["entry"], refetchType: "none" })
      qc.invalidateQueries({ queryKey: keys.counters })
      toast.success(
        filter.action === "skip"
          ? i18n.t("filters.savedRemoved", { count: updated })
          : i18n.t("filters.savedMarkedRead", { count: updated })
      )
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useSaveSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (s: Settings) => api.saveSettings(s),
    onSuccess: (s) => qc.setQueryData(keys.settings, s),
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useImportOpml() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => api.importOpml(file),
    onSuccess: (res) => {
      invalidateSubscriptions(qc)
      if (res.added > 0) {
        userRequestedRefresh = true
        qc.setQueryData<Counters>(
          keys.counters,
          (c) => c && { ...c, refreshing: true }
        )
      }
      toast.success(i18n.t("settings.importResult", res))
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}

/** Marks the browser signed in and reloads everything else. */
export function markSignedIn(qc: QueryClient, username: string) {
  qc.setQueryData<AuthStatus>(keys.auth, (s) =>
    s ? { ...s, setup_required: false, authenticated: true, username } : s
  )
  qc.invalidateQueries({ predicate: (q) => q.queryKey[0] !== keys.auth[0] })
}

/** Marks the browser signed out and drops all cached data. */
export function markSignedOut(qc: QueryClient) {
  qc.setQueryData<AuthStatus>(keys.auth, (s) =>
    s ? { ...s, authenticated: false, username: undefined } : s
  )
  qc.removeQueries({ predicate: (q) => q.queryKey[0] !== keys.auth[0] })
}

export function useAccount() {
  return useQuery({ queryKey: keys.account, queryFn: api.account })
}

export function useUpdateAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AccountInput) => api.updateAccount(input),
    onSuccess: (account, input) => {
      qc.setQueryData(keys.account, account)
      qc.setQueryData<AuthStatus>(
        keys.auth,
        (s) => s && { ...s, username: account.username }
      )
      if (input.new_password) {
        qc.invalidateQueries({ queryKey: keys.sessions })
      }
      toast.success(
        i18n.t(input.new_password ? "account.passwordChanged" : "account.saved")
      )
    },
  })
}

export function useSessions(enabled = true) {
  return useQuery({
    queryKey: keys.sessions,
    queryFn: api.sessions,
    enabled,
  })
}

export function useRevokeSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.revokeSession(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.sessions }),
    onError: (err) => toast.error(errorMessage(err)),
  })
}

export function useRevokeOtherSessions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.revokeOtherSessions(),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: keys.sessions })
      toast.success(i18n.t("sessions.revoked", { count: res.revoked }))
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
}
