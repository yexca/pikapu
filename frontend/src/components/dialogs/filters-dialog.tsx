import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import {
  CheckCheckIcon,
  EyeOffIcon,
  ListFilterIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { errorMessage } from "@/i18n/errors"
import {
  useApplyFilter,
  useDeleteFilter,
  useFeeds,
  useFilters,
  useSaveFilter,
} from "@/lib/queries"
import type { Filter, FilterAction } from "@/lib/types"

import { useDialogs } from "./dialogs-provider"

const ALL_FEEDS = "all"

/** How the dialog opens: the list, or a new filter for one feed. */
export interface FiltersTarget {
  feedId: number | null
  /** Changes on every open so the dialog starts fresh. */
  nonce: number
}

type View =
  | { kind: "list" }
  | { kind: "form"; filter: Filter | null; feedId: number | null }

export function FiltersDialog({
  target,
  open,
  onOpenChange,
}: {
  target: FiltersTarget
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg"
        onOpenAutoFocus={(e) => {
          // Start in the keywords field rather than the first control.
          const field = (
            e.currentTarget as HTMLElement
          ).querySelector<HTMLElement>("[data-autofocus]")
          if (field) {
            e.preventDefault()
            field.focus()
          }
        }}
      >
        <FiltersBody
          key={target.nonce}
          target={target}
          onClose={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  )
}

function FiltersBody({
  target,
  onClose,
}: {
  target: FiltersTarget
  onClose: () => void
}) {
  const startWithForm = target.feedId !== null
  const [view, setView] = useState<View>(
    startWithForm
      ? { kind: "form", filter: null, feedId: target.feedId }
      : { kind: "list" }
  )
  const showList = () => setView({ kind: "list" })

  if (view.kind === "form") {
    return (
      <FilterForm
        // Remount when switching between filters so fields reset.
        key={view.filter?.id ?? "new"}
        filter={view.filter}
        feedId={view.feedId}
        onCancel={startWithForm && !view.filter ? onClose : showList}
        onSaved={showList}
      />
    )
  }
  return (
    <FilterList
      onAdd={() => setView({ kind: "form", filter: null, feedId: null })}
      onEdit={(filter) =>
        setView({ kind: "form", filter, feedId: filter.feed_id })
      }
    />
  )
}

function FilterList({
  onAdd,
  onEdit,
}: {
  onAdd: () => void
  onEdit: (filter: Filter) => void
}) {
  const { t } = useTranslation()
  const dialogs = useDialogs()
  const filters = useFilters()
  const { data: feeds = [] } = useFeeds()
  const remove = useDeleteFilter()

  const feedTitle = (id: number | null) =>
    id === null
      ? t("filters.allFeeds")
      : (feeds.find((f) => f.id === id)?.title ?? "…")

  const onDelete = async (filter: Filter) => {
    const ok = await dialogs.confirm({
      title: t("filters.deleteTitle"),
      description: t("filters.deleteDescription"),
      confirmText: t("common.delete"),
      destructive: true,
    })
    if (ok) {
      remove.mutate(filter.id, {
        onSuccess: () => toast.success(t("filters.deleted")),
      })
    }
  }

  return (
    <div className="grid gap-4">
      <DialogHeader>
        <DialogTitle>{t("filters.title")}</DialogTitle>
        <DialogDescription>{t("filters.description")}</DialogDescription>
      </DialogHeader>

      {filters.isPending ? (
        <div className="grid gap-2">
          <Skeleton className="h-14" />
          <Skeleton className="h-14" />
        </div>
      ) : filters.isError ? (
        <p className="text-xs text-destructive">
          {errorMessage(filters.error)}
        </p>
      ) : filters.data.length === 0 ? (
        <Empty className="border py-8">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ListFilterIcon />
            </EmptyMedia>
            <EmptyTitle>{t("filters.empty")}</EmptyTitle>
            <EmptyDescription>{t("filters.emptyDescription")}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ul className="divide-y rounded-lg border">
          {filters.data.map((filter) => (
            <li
              key={filter.id}
              className="flex items-start gap-2 py-2.5 pr-2 pl-3"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5 text-sm font-medium">
                  {filter.action === "skip" ? (
                    <EyeOffIcon className="size-3.5 shrink-0 text-muted-foreground" />
                  ) : (
                    <CheckCheckIcon className="size-3.5 shrink-0 text-muted-foreground" />
                  )}
                  <span className="shrink-0">
                    {filter.action === "skip"
                      ? t("filters.actionSkip")
                      : t("filters.actionMarkRead")}
                  </span>
                  <span className="truncate font-normal text-muted-foreground">
                    · {feedTitle(filter.feed_id)}
                  </span>
                </div>
                <div className="mt-1.5 flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
                  <span className="mr-0.5">{conditionLabel(t, filter)}</span>
                  {filter.keywords.map((k) => (
                    <span
                      key={k}
                      className="rounded-md bg-muted px-1.5 py-0.5 text-foreground"
                    >
                      {k}
                    </span>
                  ))}
                </div>
              </div>
              <Button
                size="icon-sm"
                variant="ghost"
                aria-label={t("filters.edit")}
                onClick={() => onEdit(filter)}
              >
                <PencilIcon />
              </Button>
              <Button
                size="icon-sm"
                variant="ghost"
                aria-label={t("filters.delete")}
                onClick={() => onDelete(filter)}
              >
                <Trash2Icon />
              </Button>
            </li>
          ))}
        </ul>
      )}

      <DialogFooter>
        <Button onClick={onAdd}>
          <PlusIcon />
          {t("filters.add")}
        </Button>
      </DialogFooter>
    </div>
  )
}

function conditionLabel(
  t: ReturnType<typeof useTranslation>["t"],
  f: Pick<Filter, "match_content" | "invert">
) {
  if (f.match_content) {
    return f.invert
      ? t("filters.summaryContentNone")
      : t("filters.summaryContent")
  }
  return f.invert ? t("filters.summaryTitleNone") : t("filters.summaryTitle")
}

/** Splits user input on commas (including CJK ones) and line breaks. */
function parseKeywords(text: string): string[] {
  return text
    .split(/[,，、\n]/)
    .map((k) => k.trim())
    .filter(Boolean)
}

function FilterForm({
  filter,
  feedId,
  onCancel,
  onSaved,
}: {
  filter: Filter | null
  feedId: number | null
  onCancel: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const id = useId()
  const { data: feeds = [] } = useFeeds()
  const save = useSaveFilter()
  const apply = useApplyFilter()

  const [feed, setFeed] = useState(feedId === null ? ALL_FEEDS : String(feedId))
  const [keywords, setKeywords] = useState(filter?.keywords.join(", ") ?? "")
  const [matchContent, setMatchContent] = useState(
    filter?.match_content ?? false
  )
  const [invert, setInvert] = useState(filter?.invert ?? false)
  const [action, setAction] = useState<FilterAction>(
    filter?.action ?? "mark_read"
  )
  const [applyNow, setApplyNow] = useState(!filter)

  const parsed = parseKeywords(keywords)
  const busy = save.isPending

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (parsed.length === 0 || busy) return
    save.mutate(
      {
        id: filter?.id,
        feed_id: feed === ALL_FEEDS ? null : Number(feed),
        keywords: parsed,
        match_content: matchContent,
        invert,
        action,
      },
      {
        onSuccess: (saved) => {
          if (applyNow) apply.mutate(saved)
          else toast.success(t("filters.saved"))
          onSaved()
        },
      }
    )
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>
          {filter ? t("filters.editTitle") : t("filters.newTitle")}
        </DialogTitle>
        <DialogDescription>{t("filters.description")}</DialogDescription>
      </DialogHeader>

      <div className="grid gap-2">
        <Label htmlFor={`${id}-feed`}>{t("filters.feed")}</Label>
        <Select value={feed} onValueChange={setFeed}>
          <SelectTrigger id={`${id}-feed`} className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_FEEDS}>{t("filters.allFeeds")}</SelectItem>
            {feeds.length > 0 && <SelectSeparator />}
            {feeds.map((f) => (
              <SelectItem key={f.id} value={String(f.id)}>
                {f.title}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor={`${id}-keywords`}>{t("filters.keywords")}</Label>
        <Textarea
          id={`${id}-keywords`}
          autoFocus
          data-autofocus
          rows={2}
          value={keywords}
          placeholder={t("filters.keywordsPlaceholder")}
          onChange={(e) => setKeywords(e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          {t("filters.keywordsHint")}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-2">
          <Label>{t("filters.lookIn")}</Label>
          <ToggleGroup
            type="single"
            variant="outline"
            size="sm"
            className="w-full"
            value={matchContent ? "content" : "title"}
            onValueChange={(v) => v && setMatchContent(v === "content")}
          >
            <ToggleGroupItem value="title" className="flex-1">
              {t("filters.lookInTitle")}
            </ToggleGroupItem>
            <ToggleGroupItem value="content" className="flex-1">
              {t("filters.lookInContent")}
            </ToggleGroupItem>
          </ToggleGroup>
        </div>
        <div className="grid gap-2">
          <Label htmlFor={`${id}-condition`}>{t("filters.condition")}</Label>
          <Select
            value={invert ? "none" : "any"}
            onValueChange={(v) => setInvert(v === "none")}
          >
            <SelectTrigger id={`${id}-condition`} size="sm" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="any">{t("filters.conditionAny")}</SelectItem>
              <SelectItem value="none">{t("filters.conditionNone")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid gap-2">
        <Label>{t("filters.action")}</Label>
        <ToggleGroup
          type="single"
          variant="outline"
          size="sm"
          className="w-full sm:w-1/2"
          value={action}
          onValueChange={(v) => v && setAction(v as FilterAction)}
        >
          <ToggleGroupItem value="mark_read" className="flex-1">
            <CheckCheckIcon />
            {t("filters.actionMarkRead")}
          </ToggleGroupItem>
          <ToggleGroupItem value="skip" className="flex-1">
            <EyeOffIcon />
            {t("filters.actionSkip")}
          </ToggleGroupItem>
        </ToggleGroup>
        {action === "skip" && (
          <p className="text-xs text-muted-foreground">
            {t("filters.actionSkipHint")}
          </p>
        )}
      </div>

      <div className="flex items-center justify-between gap-4 rounded-lg border px-3 py-2.5">
        <div className="min-w-0">
          <Label htmlFor={`${id}-apply`}>{t("filters.applyNow")}</Label>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {t("filters.applyNowHint")}
          </p>
        </div>
        <Switch
          id={`${id}-apply`}
          checked={applyNow}
          onCheckedChange={setApplyNow}
        />
      </div>

      {save.isError && (
        <p className="text-xs text-destructive">{errorMessage(save.error)}</p>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={parsed.length === 0 || busy}>
          {busy && <Spinner />}
          {t("common.save")}
        </Button>
      </DialogFooter>
    </form>
  )
}
