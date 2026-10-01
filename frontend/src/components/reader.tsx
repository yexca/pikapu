import { useTranslation } from "react-i18next"
import { Link } from "react-router"
import { toast } from "sonner"
import {
  ArrowLeftIcon,
  ArrowUpRightIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  CircleIcon,
  CircleDotIcon,
  ExternalLinkIcon,
  Link2Icon,
  StarIcon,
  XIcon,
} from "lucide-react"

import { FeedIcon } from "@/components/feed-icon"
import { Mascot } from "@/components/mascot"
import { Button } from "@/components/ui/button"
import { Kbd } from "@/components/ui/kbd"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { errorMessage } from "@/i18n/errors"
import { usePrefs } from "@/lib/prefs"
import { useEntry } from "@/lib/queries"
import { fullTime } from "@/lib/time"
import type { Entry, Feed } from "@/lib/types"
import { cn } from "@/lib/utils"

const proseSize = {
  sm: "prose-sm",
  base: "prose-base",
  lg: "prose-lg",
}

interface ReaderProps {
  entry: Entry | null
  feed?: Feed
  onClose: () => void
  onPrev?: () => void
  onNext?: () => void
  onToggleRead: () => void
  onToggleStar: () => void
  /** Shown in the hub's side sheet rather than as a pane. */
  inSheet?: boolean
}

function ToolbarButton({
  label,
  onClick,
  disabled,
  className,
  children,
}: {
  label: string
  onClick?: () => void
  disabled?: boolean
  className?: string
  children: React.ReactNode
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={label}
          onClick={onClick}
          disabled={disabled}
          className={className}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

export function Reader({
  entry,
  feed,
  onClose,
  onPrev,
  onNext,
  onToggleRead,
  onToggleStar,
  inSheet = false,
}: ReaderProps) {
  const { t } = useTranslation()
  const detail = useEntry(entry?.id ?? null)
  const [prefs] = usePrefs()

  if (!entry) return <ReaderEmpty />

  const content = detail.data?.content
  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(entry.url)
      toast.success(t("reader.linkCopied"))
    } catch {
      toast.error(t("reader.copyFailed"))
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-12 shrink-0 items-center gap-0.5 border-b px-2 lg:px-3">
        <ToolbarButton
          label={t("reader.back")}
          onClick={onClose}
          className={inSheet ? "sm:hidden" : "lg:hidden"}
        >
          <ArrowLeftIcon />
        </ToolbarButton>
        <ToolbarButton
          label={t("reader.previous")}
          onClick={onPrev}
          disabled={!onPrev}
        >
          <ChevronUpIcon />
        </ToolbarButton>
        <ToolbarButton
          label={t("reader.next")}
          onClick={onNext}
          disabled={!onNext}
        >
          <ChevronDownIcon />
        </ToolbarButton>

        <div className="ml-auto flex items-center gap-0.5">
          <ToolbarButton
            label={entry.is_starred ? t("reader.unstar") : t("reader.star")}
            onClick={onToggleStar}
          >
            <StarIcon
              className={cn(
                entry.is_starred && "fill-amber-400 text-amber-400"
              )}
            />
          </ToolbarButton>
          <ToolbarButton
            label={
              entry.is_read ? t("reader.markUnread") : t("reader.markRead")
            }
            onClick={onToggleRead}
          >
            {entry.is_read ? (
              <CircleIcon />
            ) : (
              <CircleDotIcon className="text-brand" />
            )}
          </ToolbarButton>
          {entry.url && (
            <>
              <ToolbarButton label={t("reader.copyLink")} onClick={copyLink}>
                <Link2Icon />
              </ToolbarButton>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button size="icon-sm" variant="ghost" asChild>
                    <a
                      href={entry.url}
                      target="_blank"
                      rel="noreferrer"
                      aria-label={t("reader.openOriginal")}
                    >
                      <ExternalLinkIcon />
                    </a>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("reader.openOriginal")}</TooltipContent>
              </Tooltip>
            </>
          )}
          {inSheet && (
            <ToolbarButton
              label={t("reader.close")}
              onClick={onClose}
              className="max-sm:hidden"
            >
              <XIcon />
            </ToolbarButton>
          )}
        </div>
      </div>

      <div
        key={entry.id}
        className="scroll-thin min-h-0 flex-1 overflow-y-auto overscroll-contain pb-[env(safe-area-inset-bottom)]"
      >
        <article className="mx-auto w-full max-w-[46rem] px-5 pt-8 pb-16 sm:px-8 lg:pt-12">
          <div className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
            <FeedIcon feed={feed} />
            {feed ? (
              <Link
                to={`/feeds/${feed.id}`}
                className="min-w-0 truncate font-medium text-foreground/80 transition-colors hover:text-foreground"
              >
                {feed.title}
              </Link>
            ) : (
              <span>…</span>
            )}
            {entry.author && (
              <>
                <span className="opacity-60">·</span>
                <span className="min-w-0 truncate">{entry.author}</span>
              </>
            )}
          </div>

          <h1 className="text-2xl leading-tight font-semibold tracking-tight text-balance break-words sm:text-[1.75rem]">
            {entry.url ? (
              <a
                href={entry.url}
                target="_blank"
                rel="noreferrer"
                className="decoration-brand/50 underline-offset-4 hover:underline"
              >
                {entry.title || t("common.untitled")}
              </a>
            ) : (
              entry.title || t("common.untitled")
            )}
          </h1>
          <time
            dateTime={entry.published_at}
            className="mt-3 block text-sm text-muted-foreground"
          >
            {fullTime(entry.published_at)}
          </time>

          <div className="my-8 h-px bg-border" />

          {detail.isPending ? (
            <ContentSkeleton />
          ) : detail.isError ? (
            <div className="flex flex-col items-center gap-4 py-6 text-center">
              <Mascot pose="oops" className="h-32" />
              <p className="text-sm text-destructive">
                {t("reader.loadFailed", { reason: errorMessage(detail.error) })}
              </p>
            </div>
          ) : content ? (
            <div
              className={cn(
                "article prose max-w-none prose-neutral dark:prose-invert",
                proseSize[prefs.fontSize]
              )}
              dangerouslySetInnerHTML={{ __html: content }}
            />
          ) : (
            <p className="text-sm text-muted-foreground">
              {t("reader.noContent")}
            </p>
          )}

          {/* End of the article: the way out to the site, and the mascot
              settling down with the letter she delivered. */}
          {!detail.isPending && (
            <div className="mt-14 flex items-end justify-between gap-4 border-t pt-6">
              {entry.url ? (
                <Button variant="outline" asChild>
                  <a href={entry.url} target="_blank" rel="noreferrer">
                    {t("reader.readOriginal")}
                    <ArrowUpRightIcon />
                  </a>
                </Button>
              ) : (
                <span />
              )}
              {!detail.isError && (
                <Mascot pose="reading" className="-mb-4 h-24 sm:h-28" />
              )}
            </div>
          )}
        </article>
      </div>
    </div>
  )
}

function ContentSkeleton() {
  return (
    <div className="space-y-3">
      {[100, 96, 90, 98, 60].map((w, i) => (
        <Skeleton key={i} className="h-4" style={{ width: `${w}%` }} />
      ))}
      <Skeleton className="!mt-8 h-48 w-full" />
    </div>
  )
}

function ReaderEmpty() {
  const { t } = useTranslation()
  return (
    <div className="flex h-full flex-col items-center justify-center gap-5 p-8 text-center">
      {/* Only shown beside the classic list, so she points at it. */}
      <Mascot pose="pointing" className="h-44" />
      <div>
        <p className="text-sm font-medium">{t("reader.emptyTitle")}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("reader.emptySubtitle")}
        </p>
      </div>
      <div className="mt-2 flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Kbd>J</Kbd>
          <Kbd>K</Kbd>
          {t("reader.hintNavigate")}
        </span>
        <span className="flex items-center gap-1.5">
          <Kbd>M</Kbd>
          {t("reader.hintRead")}
        </span>
        <span className="flex items-center gap-1.5">
          <Kbd>S</Kbd>
          {t("reader.hintStar")}
        </span>
        <span className="flex items-center gap-1.5">
          <Kbd>?</Kbd>
          {t("reader.hintShortcuts")}
        </span>
      </div>
    </div>
  )
}
