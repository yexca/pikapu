import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router"
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
import { errorMessage } from "@/i18n/errors"
import { useAddFeed } from "@/lib/queries"
import { useView } from "@/lib/view"

import {
  CategoryField,
  categoryPayload,
  NEW_CATEGORY,
  NO_CATEGORY,
  type CategoryChoice,
} from "./category-field"

export function AddFeedDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <AddFeedForm onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}

function AddFeedForm({ onDone }: { onDone: () => void }) {
  const { t } = useTranslation()
  const view = useView()
  const navigate = useNavigate()
  const add = useAddFeed()
  const [url, setUrl] = useState("")
  const [category, setCategory] = useState<CategoryChoice>({
    // Default to the category currently being browsed.
    value: view.kind === "category" ? String(view.id) : NO_CATEGORY,
    newName: "",
  })

  const invalid =
    !url.trim() || (category.value === NEW_CATEGORY && !category.newName.trim())

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid || add.isPending) return
    add.mutate(
      { url: url.trim(), ...categoryPayload(category) },
      {
        onSuccess: (feed) => {
          toast.success(t("addFeed.added", { title: feed.title }))
          onDone()
          navigate(`/feeds/${feed.id}`)
        },
      }
    )
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>{t("addFeed.title")}</DialogTitle>
        <DialogDescription>{t("addFeed.description")}</DialogDescription>
      </DialogHeader>

      <div className="grid gap-2">
        <Label htmlFor="feed-url">{t("addFeed.url")}</Label>
        <Input
          id="feed-url"
          autoFocus
          inputMode="url"
          autoComplete="off"
          placeholder="https://example.com/feed.xml"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          aria-invalid={add.isError || undefined}
        />
        {add.isError && (
          <p className="text-xs text-destructive">{errorMessage(add.error)}</p>
        )}
      </div>

      <div className="grid gap-2">
        <Label htmlFor="feed-category">{t("addFeed.category")}</Label>
        <CategoryField
          id="feed-category"
          choice={category}
          onChange={setCategory}
        />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={invalid || add.isPending}>
          {add.isPending && <Spinner />}
          {add.isPending ? t("addFeed.submitting") : t("addFeed.submit")}
        </Button>
      </DialogFooter>
    </form>
  )
}
