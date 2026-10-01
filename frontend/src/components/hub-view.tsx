import { memo, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  ChevronDownIcon,
  ChevronUpIcon,
  GemIcon,
  HeartIcon,
  InboxIcon,
  SparklesIcon,
  StarIcon,
  ZapIcon,
} from "lucide-react"

import {
  ListEmpty,
  ListToolbar,
  type EntryListProps,
} from "@/components/entry-list"
import { FeedIcon } from "@/components/feed-icon"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { useTick } from "@/hooks/use-debounced"
import { useLoadMore } from "@/hooks/use-load-more"
import type { DayGroup, Stack } from "@/lib/hub"
import { dayLabel, relativeTime } from "@/lib/time"
import type { Entry, Feed, PickReason } from "@/lib/types"
import { cn } from "@/lib/utils"

/** Entries a stack shows before "Show more". */
const STACK_PREVIEW = 3
/** Picks shown as cards (the first as the hero); the rest are listed. */
const PICK_CARDS = 5

export interface HubViewProps extends EntryListProps {
  /** Recommended entries, shown above the stream; empty to hide the section. */
  picks: Entry[]
  /** The stream: `entries` grouped by day and feed. */
  days: DayGroup[]
}

type Select = (entry: Entry) => void

/**
 * The hub layout: a centered, message-center style stream with recommended
 * picks first and newer updates grouped by day and source.
 */
export function HubView(props: HubViewProps) {
  const {
    picks,
    days,
    entries,
    feedsById,
    selectedId,
    onSelect,
    showFeed,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = props
  useTick(60_000)
  const { t } = useTranslation()
  const sentinelRef = useRef<HTMLDivElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const total = picks.length + entries.length

  useLoadMore(sentinelRef, scrollRef, {
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    itemCount: entries.length,
  })

  // Keep the selected entry visible during keyboard navigation.
  useEffect(() => {
    if (selectedId == null) return
    scrollRef.current
      ?.querySelector(`[data-entry-id="${selectedId}"]`)
      ?.scrollIntoView({ block: "nearest" })
  }, [selectedId])

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListToolbar {...props} centered />
      <div
        ref={scrollRef}
        className="scroll-thin min-h-0 flex-1 overflow-y-auto overscroll-contain bg-muted/40 pb-[env(safe-area-inset-bottom)]"
      >
        {props.isLoading ? (
          <HubSkeleton />
        ) : total === 0 ? (
          <ListEmpty kind={props.emptyKind} query={props.query} />
        ) : (
          <div className="mx-auto w-full max-w-3xl px-3 pt-5 sm:px-6 sm:pt-8">
            {picks.length > 0 && (
              <Picks
                picks={picks}
                feedsById={feedsById}
                selectedId={selectedId}
                onSelect={onSelect}
              />
            )}
            {days.length > 0 && (
              <section className={cn(picks.length > 0 && "mt-10")}>
                <SectionTitle icon={<InboxIcon />} title={t("hub.latest")} />
                {days.map((day) => (
                  <div key={day.key} className="mt-4 first-of-type:mt-0">
                    <h3 className="pointer-events-none sticky top-2 z-10 mb-2.5 w-fit rounded-full border bg-background/90 px-3 py-1 text-xs font-medium text-muted-foreground shadow-xs backdrop-blur">
                      {dayLabel(day.date)}
                    </h3>
                    <div className="space-y-2.5">
                      {day.stacks.map((stack) => (
                        <StackCard
                          key={stack.key}
                          stack={stack}
                          feed={
                            stack.feedId != null
                              ? feedsById.get(stack.feedId)
                              : undefined
                          }
                          grouped={showFeed}
                          selectedId={selectedId}
                          onSelect={onSelect}
                        />
                      ))}
                    </div>
                  </div>
                ))}
              </section>
            )}
          </div>
        )}
        <div ref={sentinelRef} />
        {isFetchingNextPage && (
          <div className="flex justify-center py-4">
            <Spinner className="text-muted-foreground" />
          </div>
        )}
        {!props.isLoading && !hasNextPage && total > 0 && (
          <p className="py-8 text-center text-xs text-muted-foreground/70">
            {t("list.end")}
          </p>
        )}
      </div>
    </div>
  )
}

function SectionTitle({
  icon,
  title,
  hint,
}: {
  icon: React.ReactNode
  title: string
  hint?: string
}) {
  return (
    <div className="mb-3 flex min-w-0 items-baseline gap-2 px-1">
      <h2 className="flex shrink-0 items-center gap-1.5 self-center text-sm font-semibold [&_svg]:size-4 [&_svg]:text-brand">
        {icon}
        {title}
      </h2>
      {hint && (
        <p className="min-w-0 truncate text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  )
}

function summaryOf(entry: Entry) {
  return entry.summary && !entry.summary.startsWith(entry.title)
    ? entry.summary
    : ""
}

function ReasonBadge({
  reason,
  className,
}: {
  reason?: PickReason
  className?: string
}) {
  const { t } = useTranslation()
  const icon =
    reason &&
    {
      favorite_source: <HeartIcon />,
      rare_source: <GemIcon />,
      fresh: <ZapIcon />,
    }[reason]
  if (!reason || !icon) return null
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-full bg-brand/10 px-2 py-0.5 text-[11px] leading-none font-medium text-brand [&_svg]:size-3",
        className
      )}
    >
      {icon}
      {t(`hub.reasons.${reason}`)}
    </span>
  )
}

function StateMarks({ entry }: { entry: Entry }) {
  return (
    <>
      {entry.is_starred && (
        <StarIcon className="size-3.5 shrink-0 fill-amber-400 text-amber-400" />
      )}
      {!entry.is_read && (
        <span className="size-2 shrink-0 rounded-full bg-brand" />
      )}
    </>
  )
}

function Thumb({
  src,
  dim,
  className,
}: {
  src: string
  dim?: boolean
  className?: string
}) {
  const [failed, setFailed] = useState(false)
  if (!src || failed) return null
  return (
    <img
      src={src}
      alt=""
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
      className={cn(
        "shrink-0 bg-muted object-cover",
        dim && "opacity-70",
        className
      )}
    />
  )
}

/** Feed, time, and read/star marks above a pick's title. */
function PickMeta({ entry, feed }: { entry: Entry; feed?: Feed }) {
  return (
    <div className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
      <FeedIcon feed={feed} className="size-3.5" />
      <span className="min-w-0 truncate">{feed?.title ?? "…"}</span>
      <span className="shrink-0 opacity-60">·</span>
      <time className="shrink-0" dateTime={entry.published_at}>
        {relativeTime(entry.published_at)}
      </time>
      <span className="ml-auto flex shrink-0 items-center gap-1.5 pl-1">
        <StateMarks entry={entry} />
      </span>
    </div>
  )
}

const cardClass =
  "w-full overflow-hidden border bg-card text-left shadow-xs transition outline-none hover:border-foreground/15 hover:shadow-md focus-visible:ring-2 focus-visible:ring-ring/50"

function Picks({
  picks,
  feedsById,
  selectedId,
  onSelect,
}: {
  picks: Entry[]
  feedsById: Map<number, Feed>
  selectedId: number | null
  onSelect: Select
}) {
  const { t } = useTranslation()
  const [hero, ...rest] = picks
  const cards = rest.slice(0, PICK_CARDS - 1)
  const more = rest.slice(PICK_CARDS - 1)

  return (
    <section>
      <SectionTitle
        icon={<SparklesIcon />}
        title={t("hub.forYou")}
        hint={t("hub.forYouHint")}
      />
      <HeroCard
        entry={hero}
        feed={feedsById.get(hero.feed_id)}
        selected={hero.id === selectedId}
        onSelect={onSelect}
      />
      {cards.length > 0 && (
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          {cards.map((e) => (
            <PickCard
              key={e.id}
              entry={e}
              feed={feedsById.get(e.feed_id)}
              selected={e.id === selectedId}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
      {more.length > 0 && (
        <ul className="mt-3 divide-y divide-border/60 overflow-hidden rounded-xl border bg-card shadow-xs">
          {more.map((e) => (
            <li key={e.id}>
              <PickRow
                entry={e}
                feed={feedsById.get(e.feed_id)}
                selected={e.id === selectedId}
                onSelect={onSelect}
              />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

interface ItemProps {
  entry: Entry
  feed?: Feed
  selected: boolean
  onSelect: Select
}

function HeroCard({ entry, feed, selected, onSelect }: ItemProps) {
  const { t } = useTranslation()
  const [imageFailed, setImageFailed] = useState(false)
  const summary = summaryOf(entry)
  const image = entry.image_url && !imageFailed

  return (
    <button
      type="button"
      data-entry-id={entry.id}
      onClick={() => onSelect(entry)}
      className={cn(
        cardClass,
        "block rounded-2xl",
        selected && "border-brand/60 ring-2 ring-brand/40"
      )}
    >
      {image ? (
        <img
          src={entry.image_url}
          alt=""
          referrerPolicy="no-referrer"
          onError={() => setImageFailed(true)}
          className={cn(
            "aspect-[2/1] w-full bg-muted object-cover",
            entry.is_read && "opacity-70"
          )}
        />
      ) : (
        <div className="h-1.5 bg-linear-to-r from-brand/80 via-brand/30 to-transparent" />
      )}
      <div className="p-4 sm:p-5">
        <PickMeta entry={entry} feed={feed} />
        <h3
          className={cn(
            "mt-2 line-clamp-3 text-lg leading-snug font-semibold tracking-tight text-balance break-words sm:text-xl",
            entry.is_read && "text-muted-foreground"
          )}
        >
          {entry.title || t("common.untitled")}
        </h3>
        {summary && (
          <p className="mt-2 line-clamp-3 text-sm leading-relaxed break-words text-muted-foreground">
            {summary}
          </p>
        )}
        <ReasonBadge reason={entry.reason} className="mt-3" />
      </div>
    </button>
  )
}

function PickCard({ entry, feed, selected, onSelect }: ItemProps) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      data-entry-id={entry.id}
      onClick={() => onSelect(entry)}
      className={cn(
        cardClass,
        "flex gap-3 rounded-xl p-3.5",
        selected && "border-brand/60 ring-2 ring-brand/40"
      )}
    >
      <div className="flex min-w-0 flex-1 flex-col">
        <PickMeta entry={entry} feed={feed} />
        <h3
          className={cn(
            "mt-1.5 line-clamp-3 text-sm leading-snug break-words",
            entry.is_read
              ? "text-muted-foreground"
              : "font-medium text-foreground"
          )}
        >
          {entry.title || t("common.untitled")}
        </h3>
        <ReasonBadge reason={entry.reason} className="mt-2 self-start" />
      </div>
      <Thumb
        src={entry.image_url}
        dim={entry.is_read}
        className="mt-5 size-16 rounded-lg"
      />
    </button>
  )
}

function PickRow({ entry, feed, selected, onSelect }: ItemProps) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      data-entry-id={entry.id}
      onClick={() => onSelect(entry)}
      className={cn(
        "relative flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm transition-colors outline-none hover:bg-muted/50 focus-visible:bg-muted/60",
        selected && "bg-muted hover:bg-muted"
      )}
    >
      {selected && <SelectedBar />}
      <FeedIcon feed={feed} />
      <span
        className={cn(
          "min-w-0 flex-1 truncate",
          entry.is_read ? "text-muted-foreground" : "font-medium"
        )}
      >
        {entry.title || t("common.untitled")}
      </span>
      <ReasonBadge reason={entry.reason} className="max-sm:hidden" />
      <StateMarks entry={entry} />
      <time
        dateTime={entry.published_at}
        className="shrink-0 text-xs text-muted-foreground"
      >
        {relativeTime(entry.published_at)}
      </time>
    </button>
  )
}

function SelectedBar() {
  return (
    <span className="absolute inset-y-2 left-0 w-[3px] rounded-r-full bg-brand" />
  )
}

/**
 * A notification-style card: one feed's updates for a day (grouped), or a
 * single entry. Long stacks collapse behind "Show more", but expand while
 * keyboard navigation selects a hidden entry.
 */
const StackCard = memo(function StackCard({
  stack,
  feed,
  grouped,
  selectedId,
  onSelect,
}: {
  stack: Stack
  feed?: Feed
  grouped: boolean
  selectedId: number | null
  onSelect: Select
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { entries } = stack
  const selectedIndex = entries.findIndex((e) => e.id === selectedId)
  const expanded = open || selectedIndex >= STACK_PREVIEW
  const visible = expanded ? entries : entries.slice(0, STACK_PREVIEW)
  const hidden = entries.length - visible.length
  const unread = entries.filter((e) => !e.is_read).length
  const [lead, ...rest] = visible

  return (
    <div className="overflow-hidden rounded-xl border bg-card shadow-xs">
      {grouped && (
        <div className="flex items-center gap-2 px-4 pt-3 text-xs text-muted-foreground">
          <FeedIcon feed={feed} />
          <span className="min-w-0 truncate font-medium text-foreground/80">
            {feed?.title ?? "…"}
          </span>
          {unread > 0 && (
            <span className="shrink-0 rounded-full bg-brand/10 px-1.5 py-px text-[11px] font-medium text-brand">
              {t("hub.newCount", { count: unread })}
            </span>
          )}
          <time dateTime={lead.published_at} className="ml-auto shrink-0 pl-1">
            {relativeTime(lead.published_at)}
          </time>
        </div>
      )}
      <ul>
        <li>
          <LeadItem
            entry={lead}
            grouped={grouped}
            selected={lead.id === selectedId}
            onSelect={onSelect}
          />
        </li>
        {rest.map((e) => (
          <li key={e.id} className="border-t border-border/50">
            <CompactItem
              entry={e}
              selected={e.id === selectedId}
              onSelect={onSelect}
            />
          </li>
        ))}
      </ul>
      {(hidden > 0 || (open && entries.length > STACK_PREVIEW)) && (
        <button
          type="button"
          onClick={() => setOpen(hidden > 0)}
          className="flex w-full items-center justify-center gap-1 border-t border-border/50 px-4 py-2 text-xs text-muted-foreground transition-colors outline-none hover:bg-muted/50 hover:text-foreground focus-visible:bg-muted/60"
        >
          {hidden > 0 ? (
            <>
              {t("hub.showMore", { count: hidden })}
              <ChevronDownIcon className="size-3.5" />
            </>
          ) : (
            <>
              {t("hub.showLess")}
              <ChevronUpIcon className="size-3.5" />
            </>
          )}
        </button>
      )}
    </div>
  )
})

function LeadItem({
  entry,
  grouped,
  selected,
  onSelect,
}: {
  entry: Entry
  grouped: boolean
  selected: boolean
  onSelect: Select
}) {
  const { t } = useTranslation()
  const summary = summaryOf(entry)
  return (
    <button
      type="button"
      data-entry-id={entry.id}
      onClick={() => onSelect(entry)}
      className={cn(
        "relative flex w-full gap-3 px-4 py-3 text-left transition-colors outline-none hover:bg-muted/50 focus-visible:bg-muted/60",
        grouped && "pt-2",
        selected && "bg-muted hover:bg-muted"
      )}
    >
      {selected && <SelectedBar />}
      <div className="min-w-0 flex-1">
        {!grouped && (
          <div className="mb-1 flex items-center gap-1.5 text-xs text-muted-foreground">
            <time dateTime={entry.published_at}>
              {relativeTime(entry.published_at)}
            </time>
            <span className="ml-auto flex items-center gap-1.5">
              <StateMarks entry={entry} />
            </span>
          </div>
        )}
        <h4
          className={cn(
            "line-clamp-2 text-[15px] leading-snug break-words",
            entry.is_read
              ? "text-muted-foreground"
              : "font-medium text-foreground"
          )}
        >
          {grouped && !entry.is_read && (
            <span className="mr-1.5 mb-0.5 inline-block size-2 rounded-full bg-brand align-middle" />
          )}
          {entry.title || t("common.untitled")}
          {grouped && entry.is_starred && (
            <StarIcon className="mb-0.5 ml-1.5 inline size-3.5 fill-amber-400 align-middle text-amber-400" />
          )}
        </h4>
        {summary && (
          <p className="mt-1 line-clamp-2 text-[13px] leading-relaxed break-words text-muted-foreground/80">
            {summary}
          </p>
        )}
      </div>
      <Thumb
        src={entry.image_url}
        dim={entry.is_read}
        className="size-16 rounded-lg"
      />
    </button>
  )
}

function CompactItem({
  entry,
  selected,
  onSelect,
}: {
  entry: Entry
  selected: boolean
  onSelect: Select
}) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      data-entry-id={entry.id}
      onClick={() => onSelect(entry)}
      className={cn(
        "relative flex w-full items-center gap-2.5 px-4 py-2 text-left text-sm transition-colors outline-none hover:bg-muted/50 focus-visible:bg-muted/60",
        selected && "bg-muted hover:bg-muted"
      )}
    >
      {selected && <SelectedBar />}
      <span
        className={cn(
          "size-1.5 shrink-0 rounded-full",
          !entry.is_read && "bg-brand"
        )}
      />
      <span
        className={cn(
          "min-w-0 flex-1 truncate",
          entry.is_read ? "text-muted-foreground" : "text-foreground"
        )}
      >
        {entry.title || t("common.untitled")}
      </span>
      {entry.is_starred && (
        <StarIcon className="size-3.5 shrink-0 fill-amber-400 text-amber-400" />
      )}
      <time
        dateTime={entry.published_at}
        className="shrink-0 text-xs text-muted-foreground"
      >
        {relativeTime(entry.published_at)}
      </time>
    </button>
  )
}

function HubSkeleton() {
  return (
    <div className="mx-auto w-full max-w-3xl space-y-3 px-3 py-5 sm:px-6 sm:py-8">
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-60 w-full rounded-2xl" />
      <div className="grid gap-3 sm:grid-cols-2">
        <Skeleton className="h-24 rounded-xl" />
        <Skeleton className="h-24 rounded-xl" />
      </div>
      <Skeleton className="!mt-10 h-4 w-28" />
      {[0, 1, 2].map((i) => (
        <Skeleton key={i} className="h-28 w-full rounded-xl" />
      ))}
    </div>
  )
}
