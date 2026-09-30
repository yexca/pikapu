import { memo, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  CheckCheckIcon,
  InboxIcon,
  MoreHorizontalIcon,
  RefreshCwIcon,
  SearchIcon,
  SearchXIcon,
  StarIcon,
  XIcon,
} from "lucide-react"

import { FeedMenu } from "@/components/app-sidebar"
import { FeedIcon } from "@/components/feed-icon"
import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useTick } from "@/hooks/use-debounced"
import { relativeTime } from "@/lib/time"
import type { Entry, Feed } from "@/lib/types"
import { cn } from "@/lib/utils"

export interface EntryListProps {
  title: string
  subtitle?: string
  feed?: Feed
  entries: Entry[]
  feedsById: Map<number, Feed>
  selectedId: number | null
  onSelect: (entry: Entry) => void
  query: string
  onQueryChange: (q: string) => void
  searchRef: React.RefObject<HTMLInputElement | null>
  unreadOnly: boolean
  onUnreadOnlyChange?: (v: boolean) => void
  refreshing: boolean
  onRefresh: () => void
  onMarkAllRead: () => void
  isLoading: boolean
  hasNextPage: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => void
  showFeed: boolean
  emptyKind: "unread" | "starred" | "search" | "none"
}

export function EntryList(props: EntryListProps) {
  const {
    entries,
    selectedId,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = props
  useTick(60_000)
  const { t } = useTranslation()
  const sentinelRef = useRef<HTMLDivElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Load the next page when the sentinel scrolls into view.
  useEffect(() => {
    const el = sentinelRef.current
    if (!el || !hasNextPage) return
    const io = new IntersectionObserver(
      ([e]) => {
        if (e.isIntersecting && !isFetchingNextPage) fetchNextPage()
      },
      { root: scrollRef.current, rootMargin: "400px" }
    )
    io.observe(el)
    return () => io.disconnect()
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, entries.length])

  // Keep the selected entry visible during keyboard navigation.
  useEffect(() => {
    if (selectedId == null) return
    scrollRef.current
      ?.querySelector(`[data-entry-id="${selectedId}"]`)
      ?.scrollIntoView({ block: "nearest" })
  }, [selectedId])

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListHeader {...props} />
      <div
        ref={scrollRef}
        className="scroll-thin min-h-0 flex-1 overflow-y-auto overscroll-contain pb-[env(safe-area-inset-bottom)]"
      >
        {props.isLoading ? (
          <ListSkeleton />
        ) : entries.length === 0 ? (
          <ListEmpty kind={props.emptyKind} query={props.query} />
        ) : (
          <ul>
            {entries.map((e) => (
              <EntryRow
                key={e.id}
                entry={e}
                feed={props.feedsById.get(e.feed_id)}
                selected={e.id === selectedId}
                showFeed={props.showFeed}
                onSelect={props.onSelect}
              />
            ))}
          </ul>
        )}
        <div ref={sentinelRef} />
        {isFetchingNextPage && (
          <div className="flex justify-center py-4">
            <Spinner className="text-muted-foreground" />
          </div>
        )}
        {!hasNextPage && entries.length > 0 && (
          <p className="py-6 text-center text-xs text-muted-foreground/70">
            {t("list.end")}
          </p>
        )}
      </div>
    </div>
  )
}

function ListHeader({
  title,
  subtitle,
  feed,
  query,
  onQueryChange,
  searchRef,
  unreadOnly,
  onUnreadOnlyChange,
  refreshing,
  onRefresh,
  onMarkAllRead,
}: EntryListProps) {
  const { t } = useTranslation()
  return (
    <header className="flex shrink-0 flex-col gap-2.5 border-b px-3 pt-2.5 pb-3">
      <div className="flex h-9 items-center gap-1">
        <SidebarTrigger className="-ml-0.5 text-muted-foreground" />
        <div className="min-w-0 flex-1 px-1">
          <h1 className="truncate text-[15px] leading-tight font-semibold">
            {title}
          </h1>
          {subtitle && (
            <p className="truncate text-xs text-muted-foreground">{subtitle}</p>
          )}
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon-sm"
              variant="ghost"
              disabled={refreshing}
              onClick={onRefresh}
              aria-label={t("list.refresh")}
            >
              <RefreshCwIcon className={cn(refreshing && "animate-spin")} />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("list.refreshHint")}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon-sm"
              variant="ghost"
              onClick={onMarkAllRead}
              aria-label={t("list.markAllRead")}
            >
              <CheckCheckIcon />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("list.markAllReadHint")}</TooltipContent>
        </Tooltip>
        {feed && (
          <FeedMenu feed={feed} active>
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label={t("feed.options")}
            >
              <MoreHorizontalIcon />
            </Button>
          </FeedMenu>
        )}
      </div>
      <div className="flex items-center gap-2">
        <SearchBox
          value={query}
          onChange={onQueryChange}
          inputRef={searchRef}
        />
        {onUnreadOnlyChange && (
          <ToggleGroup
            type="single"
            size="sm"
            variant="outline"
            value={unreadOnly ? "unread" : "all"}
            onValueChange={(v) => v && onUnreadOnlyChange(v === "unread")}
            className="shrink-0"
          >
            <ToggleGroupItem value="unread" className="px-2.5 text-xs">
              {t("list.unread")}
            </ToggleGroupItem>
            <ToggleGroupItem value="all" className="px-2.5 text-xs">
              {t("list.all")}
            </ToggleGroupItem>
          </ToggleGroup>
        )}
      </div>
    </header>
  )
}

function SearchBox({
  value,
  onChange,
  inputRef,
}: {
  value: string
  onChange: (v: string) => void
  inputRef: React.RefObject<HTMLInputElement | null>
}) {
  const { t } = useTranslation()
  // Local state keeps typing responsive with IME composition.
  const [text, setText] = useState(value)
  const composing = useRef(false)

  return (
    <InputGroup className="h-7 flex-1">
      <InputGroupAddon>
        <SearchIcon />
      </InputGroupAddon>
      <InputGroupInput
        ref={inputRef}
        placeholder={t("list.search")}
        value={text}
        className="text-[13px]"
        onChange={(e) => {
          setText(e.target.value)
          if (!composing.current) onChange(e.target.value)
        }}
        onCompositionStart={() => (composing.current = true)}
        onCompositionEnd={(e) => {
          composing.current = false
          onChange(e.currentTarget.value)
        }}
        onKeyDown={(e) => {
          if (e.key === "Escape" && text) {
            e.preventDefault()
            setText("")
            onChange("")
          }
        }}
      />
      {text && (
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            size="icon-xs"
            aria-label={t("list.clearSearch")}
            onClick={() => {
              setText("")
              onChange("")
            }}
          >
            <XIcon />
          </InputGroupButton>
        </InputGroupAddon>
      )}
    </InputGroup>
  )
}

const EntryRow = memo(function EntryRow({
  entry,
  feed,
  selected,
  showFeed,
  onSelect,
}: {
  entry: Entry
  feed?: Feed
  selected: boolean
  showFeed: boolean
  onSelect: (entry: Entry) => void
}) {
  // Also re-renders this memoized row when the language changes.
  const { t } = useTranslation()
  const [imageFailed, setImageFailed] = useState(false)
  const summary =
    entry.summary && !entry.summary.startsWith(entry.title) ? entry.summary : ""

  return (
    <li>
      <button
        type="button"
        data-entry-id={entry.id}
        onClick={() => onSelect(entry)}
        className={cn(
          "relative flex w-full gap-3 border-b border-border/60 px-4 py-3 text-left transition-colors outline-none hover:bg-muted/50 focus-visible:bg-muted/60",
          selected && "bg-muted hover:bg-muted"
        )}
      >
        {selected && (
          <span className="absolute inset-y-2 left-0 w-[3px] rounded-r-full bg-brand" />
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            {showFeed && (
              <>
                <FeedIcon feed={feed} className="size-3.5" />
                <span className="min-w-0 truncate">{feed?.title ?? "…"}</span>
                <span className="shrink-0 opacity-60">·</span>
              </>
            )}
            <time className="shrink-0" dateTime={entry.published_at}>
              {relativeTime(entry.published_at)}
            </time>
            <span className="ml-auto flex shrink-0 items-center gap-1.5 pl-1">
              {entry.is_starred && (
                <StarIcon className="size-3.5 fill-amber-400 text-amber-400" />
              )}
              {!entry.is_read && (
                <span className="size-2 rounded-full bg-brand" />
              )}
            </span>
          </div>
          <h3
            className={cn(
              "mt-1 line-clamp-2 text-sm leading-snug break-words",
              entry.is_read
                ? "text-muted-foreground"
                : "font-medium text-foreground"
            )}
          >
            {entry.title || t("common.untitled")}
          </h3>
          {summary && (
            <p className="mt-1 line-clamp-2 text-[13px] leading-relaxed break-words text-muted-foreground/80">
              {summary}
            </p>
          )}
        </div>
        {entry.image_url && !imageFailed && (
          <img
            src={entry.image_url}
            alt=""
            loading="lazy"
            referrerPolicy="no-referrer"
            onError={() => setImageFailed(true)}
            className={cn(
              "mt-5 size-16 shrink-0 rounded-md bg-muted object-cover",
              entry.is_read && "opacity-70"
            )}
          />
        )}
      </button>
    </li>
  )
})

function ListSkeleton() {
  return (
    <div>
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="border-b border-border/60 px-4 py-3.5">
          <Skeleton className="h-3 w-32" />
          <Skeleton className="mt-2.5 h-4 w-11/12" />
          <Skeleton className="mt-2 h-3 w-full" />
          <Skeleton className="mt-1.5 h-3 w-2/3" />
        </div>
      ))}
    </div>
  )
}

function ListEmpty({
  kind,
  query,
}: {
  kind: EntryListProps["emptyKind"]
  query: string
}) {
  const { t } = useTranslation()
  const content = {
    unread: {
      icon: <CheckCheckIcon />,
      title: t("list.emptyUnreadTitle"),
      description: t("list.emptyUnreadDescription"),
    },
    starred: {
      icon: <StarIcon />,
      title: t("list.emptyStarredTitle"),
      description: t("list.emptyStarredDescription"),
    },
    search: {
      icon: <SearchXIcon />,
      title: t("list.emptySearchTitle"),
      description: t("list.emptySearchDescription", { query: query.trim() }),
    },
    none: {
      icon: <InboxIcon />,
      title: t("list.emptyTitle"),
      description: t("list.emptyDescription"),
    },
  }[kind]

  return (
    <Empty className="h-full min-h-72">
      <EmptyHeader>
        <EmptyMedia variant="icon">{content.icon}</EmptyMedia>
        <EmptyTitle>{content.title}</EmptyTitle>
        <EmptyDescription>{content.description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  )
}
