import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { useDialogs } from "@/components/dialogs/dialogs-provider"
import { EntryList } from "@/components/entry-list"
import { Reader } from "@/components/reader"
import { useDebounced } from "@/hooks/use-debounced"
import { useHotkeys } from "@/hooks/use-hotkeys"
import { useIsMobile } from "@/hooks/use-mobile"
import { usePrefs } from "@/lib/prefs"
import {
  findCachedEntry,
  prefetchEntry,
  useCategories,
  useCounters,
  useEntries,
  useEntry,
  useFeeds,
  useMarkAllRead,
  useRefreshAll,
  useRefreshFeed,
  useUpdateEntry,
} from "@/lib/queries"
import type { Entry, Feed } from "@/lib/types"
import { cn } from "@/lib/utils"
import { useView, viewFilter } from "@/lib/view"

export function EntriesView() {
  const { t } = useTranslation()
  const view = useView()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const isMobile = useIsMobile()
  const dialogs = useDialogs()
  const [prefs, setPrefs] = usePrefs()
  const [params, setParams] = useSearchParams()
  const [query, setQuery] = useState("")
  const debouncedQuery = useDebounced(query, 300)
  const searchRef = useRef<HTMLInputElement>(null)

  const filter = useMemo(
    () => viewFilter(view, prefs.unreadOnly, debouncedQuery),
    [view, prefs.unreadOnly, debouncedQuery]
  )
  const entriesQuery = useEntries(filter)
  const { data: feeds } = useFeeds()
  const { data: categories } = useCategories()
  const { data: counters } = useCounters()
  const updateEntry = useUpdateEntry()
  const markAllRead = useMarkAllRead()
  const refreshAll = useRefreshAll()
  const refreshFeed = useRefreshFeed()

  const entries = useMemo(
    () => entriesQuery.data?.pages.flatMap((p) => p.entries) ?? [],
    [entriesQuery.data]
  )
  const feedsById = useMemo(
    () => new Map<number, Feed>((feeds ?? []).map((f) => [f.id, f])),
    [feeds]
  )

  const selectedId = Number(params.get("entry")) || null
  const index = entries.findIndex((e) => e.id === selectedId)
  // Fall back to the detail query when the entry isn't in the current list,
  // e.g. after a reload of a now-read entry in the unread view.
  const detail = useEntry(index < 0 ? selectedId : null)
  const selected: Entry | null =
    index >= 0 ? entries[index] : (detail.data ?? null)

  // ---- header text ----
  const currentFeed = view.kind === "feed" ? feedsById.get(view.id) : undefined
  const currentCategory =
    view.kind === "category"
      ? categories?.find((c) => c.id === view.id)
      : undefined
  const unreadIn = (list: Feed[]) =>
    list.reduce((n, f) => n + (counters?.feeds[String(f.id)] ?? 0), 0)

  let title = t("nav.all")
  let unread = counters?.unread
  if (view.kind === "starred") title = t("list.starredTitle")
  if (view.kind === "feed") {
    title = currentFeed?.title ?? ""
    unread = counters?.feeds[String(view.id)] ?? 0
  }
  if (view.kind === "category") {
    title = currentCategory?.name ?? ""
    unread = unreadIn((feeds ?? []).filter((f) => f.category_id === view.id))
  }
  const subtitle =
    view.kind === "starred"
      ? counters && t("list.articleCount", { count: counters.starred })
      : unread !== undefined
        ? unread > 0
          ? t("list.unreadCount", { count: unread })
          : t("list.noUnread")
        : undefined

  // ---- navigation ----
  // Mirrors the selection synchronously so rapid key presses (e.g. holding J)
  // don't act on a render that hasn't happened yet.
  const selectedRef = useRef(selectedId)
  useEffect(() => {
    selectedRef.current = selectedId
  }, [selectedId])

  const select = useCallback(
    (entry: Entry | null, push = false) => {
      selectedRef.current = entry?.id ?? null
      setParams(entry ? { entry: String(entry.id) } : {}, {
        replace: !(push && isMobile),
      })
    },
    [setParams, isMobile]
  )

  const close = () => {
    const idx = (window.history.state as { idx?: number } | null)?.idx ?? 0
    if (isMobile && idx > 0) navigate(-1)
    else select(null)
  }

  const { hasNextPage, isFetchingNextPage, fetchNextPage } = entriesQuery
  const move = (delta: number) => {
    if (entries.length === 0) return
    const current = entries.findIndex((e) => e.id === selectedRef.current)
    const next = current < 0 ? (delta > 0 ? 0 : -1) : current + delta
    if (next >= 0 && next < entries.length) select(entries[next])
    if (next >= entries.length - 5 && hasNextPage && !isFetchingNextPage) {
      fetchNextPage()
    }
  }

  // ---- side effects of opening an entry ----
  const autoMarked = useRef<number | null>(null)
  useEffect(() => {
    if (!selected || autoMarked.current === selected.id) return
    autoMarked.current = selected.id
    if (prefs.autoMarkRead && !selected.is_read) {
      updateEntry.mutate({ id: selected.id, patch: { is_read: true } })
    }
    // Only react to a newly selected entry, not to its state changing.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id])

  useEffect(() => {
    if (index >= 0 && index + 1 < entries.length) {
      prefetchEntry(qc, entries[index + 1].id)
    }
  }, [index, entries, qc])

  // ---- actions ----
  const current = () =>
    selectedRef.current != null
      ? findCachedEntry(qc, selectedRef.current)
      : undefined
  const toggleRead = () => {
    const e = current()
    if (e) updateEntry.mutate({ id: e.id, patch: { is_read: !e.is_read } })
  }
  const toggleStar = () => {
    const e = current()
    if (e) {
      updateEntry.mutate({ id: e.id, patch: { is_starred: !e.is_starred } })
    }
  }

  const refreshing =
    refreshFeed.isPending || refreshAll.isPending || !!counters?.refreshing
  const refresh = () => {
    if (view.kind === "feed") refreshFeed.mutate(view.id)
    else refreshAll.mutate()
  }

  const onMarkAllRead = async () => {
    const scoped = view.kind === "feed"
    if (!scoped) {
      const ok = await dialogs.confirm({
        title: t("list.confirmTitle"),
        description: filter.q
          ? t("list.confirmSearch", { query: filter.q })
          : t("list.confirmScope", { title }),
        confirmText: t("list.confirmAction"),
      })
      if (!ok) return
    }
    markAllRead.mutate(filter)
  }

  useHotkeys({
    j: () => move(1),
    k: () => move(-1),
    m: toggleRead,
    s: toggleStar,
    v: () => {
      const url = current()?.url
      if (url) window.open(url, "_blank", "noopener")
    },
    r: refresh,
    "/": () => searchRef.current?.focus(),
    A: onMarkAllRead,
    "?": dialogs.shortcuts,
    Escape: () => selectedRef.current != null && close(),
  })

  const emptyKind = debouncedQuery.trim()
    ? "search"
    : view.kind === "starred"
      ? "starred"
      : prefs.unreadOnly
        ? "unread"
        : "none"

  return (
    <div className="flex h-full min-h-0 w-full">
      <section
        className={cn(
          "flex min-h-0 w-full shrink-0 flex-col border-r lg:w-[22rem] xl:w-[26rem]",
          selected && "max-lg:hidden"
        )}
      >
        <EntryList
          title={title}
          subtitle={subtitle}
          feed={currentFeed}
          entries={entries}
          feedsById={feedsById}
          selectedId={selectedId}
          onSelect={(e) => select(e, true)}
          query={query}
          onQueryChange={setQuery}
          searchRef={searchRef}
          unreadOnly={prefs.unreadOnly}
          onUnreadOnlyChange={
            view.kind === "starred"
              ? undefined
              : (v) => setPrefs({ unreadOnly: v })
          }
          refreshing={refreshing}
          onRefresh={refresh}
          onMarkAllRead={onMarkAllRead}
          isLoading={entriesQuery.isPending}
          hasNextPage={!!hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          fetchNextPage={fetchNextPage}
          showFeed={view.kind !== "feed"}
          emptyKind={emptyKind}
        />
      </section>
      <section
        className={cn(
          "min-h-0 min-w-0 flex-1 bg-background",
          !selected && "max-lg:hidden"
        )}
      >
        <Reader
          entry={selected}
          feed={selected ? feedsById.get(selected.feed_id) : undefined}
          onClose={close}
          onPrev={index > 0 ? () => move(-1) : undefined}
          onNext={
            index >= 0 && index + 1 < entries.length ? () => move(1) : undefined
          }
          onToggleRead={toggleRead}
          onToggleStar={toggleStar}
        />
      </section>
    </div>
  )
}
