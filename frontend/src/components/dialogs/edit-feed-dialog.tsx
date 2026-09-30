import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage, feedErrorMessage } from "@/i18n/errors"
import { useUpdateFeed } from "@/lib/queries"
import { relativeTime } from "@/lib/time"
import type { Feed } from "@/lib/types"

import {
  CategoryField,
  categoryPayload,
  NEW_CATEGORY,
  NO_CATEGORY,
  type CategoryChoice,
} from "./category-field"

export function EditFeedDialog({
  feed,
  open,
  onOpenChange,
}: {
  feed: Feed | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        {feed && (
          <EditFeedForm feed={feed} onDone={() => onOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function EditFeedForm({ feed, onDone }: { feed: Feed; onDone: () => void }) {
  const { t } = useTranslation()
  const update = useUpdateFeed()
  const [title, setTitle] = useState(feed.title)
  const [feedUrl, setFeedUrl] = useState(feed.feed_url)
  const [category, setCategory] = useState<CategoryChoice>({
    value: feed.category_id ? String(feed.category_id) : NO_CATEGORY,
    newName: "",
  })

  const invalid =
    !title.trim() ||
    !feedUrl.trim() ||
    (category.value === NEW_CATEGORY && !category.newName.trim())

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid || update.isPending) return
    update.mutate(
      {
        id: feed.id,
        title: title.trim(),
        feed_url: feedUrl.trim(),
        ...categoryPayload(category),
      },
      {
        onSuccess: () => {
          toast.success(t("editFeed.saved"))
          onDone()
        },
      }
    )
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>{t("editFeed.title")}</DialogTitle>
        <DialogDescription>
          {feed.last_fetched_at
            ? t("editFeed.lastUpdated", {
                time: relativeTime(feed.last_fetched_at),
              })
            : t("editFeed.neverUpdated")}
        </DialogDescription>
      </DialogHeader>

      {feed.last_error && (
        <div className="rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">
          {t("editFeed.lastError", { reason: feedErrorMessage(feed) })}
        </div>
      )}

      <div className="grid gap-2">
        <Label htmlFor="edit-title">{t("editFeed.name")}</Label>
        <Input
          id="edit-title"
          value={title}
          maxLength={200}
          onChange={(e) => setTitle(e.target.value)}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="edit-url">{t("editFeed.url")}</Label>
        <Input
          id="edit-url"
          inputMode="url"
          value={feedUrl}
          onChange={(e) => setFeedUrl(e.target.value)}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="edit-category">{t("editFeed.category")}</Label>
        <CategoryField
          id="edit-category"
          choice={category}
          onChange={setCategory}
        />
      </div>

      {update.isError && (
        <p className="text-xs text-destructive">{errorMessage(update.error)}</p>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={invalid || update.isPending}>
          {update.isPending && <Spinner />}
          {t("common.save")}
        </Button>
      </DialogFooter>
    </form>
  )
}
