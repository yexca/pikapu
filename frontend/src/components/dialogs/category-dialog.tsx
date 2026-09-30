import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage } from "@/i18n/errors"
import { useSaveCategory } from "@/lib/queries"
import type { Category } from "@/lib/types"

export function CategoryDialog({
  category,
  open,
  onOpenChange,
}: {
  category: Category | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <CategoryForm
          key={category?.id ?? "new"}
          category={category}
          onDone={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  )
}

function CategoryForm({
  category,
  onDone,
}: {
  category: Category | null
  onDone: () => void
}) {
  const { t } = useTranslation()
  const save = useSaveCategory()
  const [name, setName] = useState(category?.name ?? "")

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || save.isPending) return
    save.mutate({ id: category?.id, name: name.trim() }, { onSuccess: onDone })
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>
          {category
            ? t("categoryDialog.renameTitle")
            : t("categoryDialog.createTitle")}
        </DialogTitle>
      </DialogHeader>
      <div className="grid gap-2">
        <Input
          autoFocus
          placeholder={t("categoryField.placeholder")}
          maxLength={64}
          value={name}
          onChange={(e) => setName(e.target.value)}
          aria-invalid={save.isError || undefined}
        />
        {save.isError && (
          <p className="text-xs text-destructive">{errorMessage(save.error)}</p>
        )}
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={!name.trim() || save.isPending}>
          {save.isPending && <Spinner />}
          {category ? t("common.save") : t("common.create")}
        </Button>
      </DialogFooter>
    </form>
  )
}
