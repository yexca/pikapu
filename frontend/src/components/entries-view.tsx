import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { AppHeader } from "@/components/app-header"
import { useDialogs } from "@/components/dialogs/dialogs-provider"
import { EntryList, type EntryListProps } from "@/components/entry-list"
import { HubView } from "@/components/hub-view"
import { Reader } from "@/components/reader"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { useDebounced } from "@/hooks/use-debounced"
import { useHotkeys } from "@/hooks/use-hotkeys"
import { useIsMobile } from "@/hooks/use-mobile"
import { flattenDays, groupByDay } from "@/lib/hub"
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
  usePicks,
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

  // ---- hub layout: picks, then the rest grouped by day and feed ----
  const hub = prefs.layout === "hub"
  const showPicks =
    hub &&
    !debouncedQuery.trim() &&
    (view.kind === "all" || view.kind === "category")
  const picksQuery = usePicks(filter, showPicks)
  const picks = useMemo(
    () =>
      showPicks ? (picksQuery.data?.pages.flatMap((p) => p.entries) ?? []) : [],
    [showPicks, picksQuery.data]
  )
  const days = useMemo(() => {
    if (!hub) return []
    const picked = new Set(picks.map((e) => e.id))
    return groupByDay(
      entries.filter((e) => !picked.has(e.id)),
      view.kind !== "feed"
    )
  }, [hub, picks, entries, view.kind])
  // Entries in display order, which is what J/K and Previous/Next follow.
  const ordered = useMemo(
    () => (hub ? [...picks, ...flattenDays(days)] : entries),
    [hub, picks, days, entries]
  )

  const selectedId = Number(params.get("entry")) || null
  const index = ordered.findIndex((e) => e.id === selectedId)
  // Fall back to the detail query when the entry isn't in the current list,
  // e.g. after a reload of a now-read entry in the unread view.
  const detail = useEntry(index < 0 ? selectedId : null)
  const selected: Entry | null =
    index >= 0 ? ordered[index] : (detail.data ?? null)

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
    if (ordered.length === 0) return
    const current = ordered.findIndex((e) => e.id === selectedRef.current)
    const next = current < 0 ? (delta > 0 ? 0 : -1) : current + delta
    if (next >= 0 && next < ordered.length) select(ordered[next])
    if (next >= ordered.length - 5 && hasNextPage && !isFetchingNextPage) {
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
    if (index >= 0 && index + 1 < ordered.length) {
      prefetchEntry(qc, ordered[index + 1].id)
    }
  }, [index, ordered, qc])

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

  const listProps: EntryListProps = {
    feed: currentFeed,
    entries: hub ? flattenDays(days) : entries,
    feedsById,
    selectedId,
    onSelect: (e) => select(e, true),
    onOpenOriginal: (e) => {
      if (prefs.autoMarkRead && !e.is_read) {
        updateEntry.mutate({ id: e.id, patch: { is_read: true } })
      }
    },
    query,
    onQueryChange: setQuery,
    searchRef,
    unreadOnly: prefs.unreadOnly,
    onUnreadOnlyChange:
      view.kind === "starred" ? undefined : (v) => setPrefs({ unreadOnly: v }),
    onMarkAllRead,
    isLoading: entriesQuery.isPending || (showPicks && picksQuery.isPending),
    hasNextPage: !!hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    showFeed: view.kind !== "feed",
    emptyKind,
  }
  const readerProps = {
    feed: selected ? feedsById.get(selected.feed_id) : undefined,
    onClose: close,
    onPrev: index > 0 ? () => move(-1) : undefined,
    onNext:
      index >= 0 && index + 1 < ordered.length ? () => move(1) : undefined,
    onToggleRead: toggleRead,
    onToggleStar: toggleStar,
  }

  const header = (
    <AppHeader
      title={title}
      subtitle={subtitle}
      refreshing={refreshing}
      onRefresh={refresh}
      // On narrow screens the classic reader replaces the whole column.
      className={cn(!hub && selected && "max-lg:hidden")}
    />
  )

  if (hub) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        {header}
        <div className="min-h-0 flex-1">
          <HubView {...listProps} picks={picks} days={days} />
        </div>
        <ReaderSheet entry={selected} {...readerProps} />
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {header}
      <div className="flex min-h-0 w-full flex-1">
        <section
          className={cn(
            "flex min-h-0 w-full shrink-0 flex-col border-r lg:w-[22rem] xl:w-[26rem]",
            selected && "max-lg:hidden"
          )}
        >
          <EntryList {...listProps} />
        </section>
        <section
          className={cn(
            "min-h-0 min-w-0 flex-1 bg-background",
            !selected && "max-lg:hidden"
          )}
        >
          <Reader entry={selected} {...readerProps} />
        </section>
      </div>
    </div>
  )
}

/**
 * The hub's reader: a sheet over the stream (full screen on phones). It
 * keeps showing the last article while it animates closed, and lets the
 * reading shortcuts through.
 */
function ReaderSheet({
  entry,
  feed,
  ...props
}: React.ComponentProps<typeof Reader>) {
  const [shown, setShown] = useState({ entry, feed })
  if (entry && (entry !== shown.entry || feed !== shown.feed)) {
    setShown({ entry, feed })
  }
  const contentRef = useRef<HTMLDivElement>(null)

  return (
    <Sheet open={!!entry} onOpenChange={(open) => !open && props.onClose()}>
      <SheetContent
        ref={contentRef}
        side="right"
        showCloseButton={false}
        data-allow-hotkeys
        aria-describedby={undefined}
        // Focusing the first toolbar button would pop its tooltip.
        onOpenAutoFocus={(e) => {
          e.preventDefault()
          contentRef.current?.focus()
        }}
        onCloseAutoFocus={(e) => e.preventDefault()}
        // Close through onClose only once; the Escape hotkey must not repeat it.
        onEscapeKeyDown={(e) => {
          e.preventDefault()
          props.onClose()
        }}
        className="gap-0 p-0 outline-none data-[side=right]:w-full data-[side=right]:sm:max-w-2xl xl:data-[side=right]:max-w-3xl"
      >
        <SheetTitle className="sr-only">{shown.entry?.title}</SheetTitle>
        <Reader {...shown} {...props} inSheet />
      </SheetContent>
    </Sheet>
  )
}
