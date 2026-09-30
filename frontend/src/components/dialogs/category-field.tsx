/* eslint-disable react-refresh/only-export-components */
import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useCategories } from "@/lib/queries"

export const NO_CATEGORY = "none"
export const NEW_CATEGORY = "new"

export interface CategoryChoice {
  /** A category id, NO_CATEGORY or NEW_CATEGORY. */
  value: string
  newName: string
}

/** Converts the picker state into the API's category fields. */
export function categoryPayload(c: CategoryChoice) {
  if (c.value === NEW_CATEGORY) {
    return { category_id: null, category_name: c.newName.trim() }
  }
  if (c.value === NO_CATEGORY) return { category_id: null }
  return { category_id: Number(c.value) }
}

export function CategoryField({
  id,
  choice,
  onChange,
}: {
  id?: string
  choice: CategoryChoice
  onChange: (c: CategoryChoice) => void
}) {
  const { t } = useTranslation()
  const { data: categories = [] } = useCategories()

  return (
    <div className="flex flex-col gap-2">
      <Select
        value={choice.value}
        onValueChange={(value) => onChange({ ...choice, value })}
      >
        <SelectTrigger id={id} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NO_CATEGORY}>{t("categoryField.none")}</SelectItem>
          {categories.map((c) => (
            <SelectItem key={c.id} value={String(c.id)}>
              {c.name}
            </SelectItem>
          ))}
          <SelectSeparator />
          <SelectItem value={NEW_CATEGORY}>{t("categoryField.new")}</SelectItem>
        </SelectContent>
      </Select>
      {choice.value === NEW_CATEGORY && (
        <Input
          autoFocus
          placeholder={t("categoryField.placeholder")}
          value={choice.newName}
          maxLength={64}
          onChange={(e) => onChange({ ...choice, newName: e.target.value })}
        />
      )}
    </div>
  )
}
