import { useTranslation } from "react-i18next"

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Kbd } from "@/components/ui/kbd"
import type { Messages } from "@/i18n/locales/en"

const shortcuts: [string[], keyof Messages["shortcuts"]][] = [
  [["J"], "next"],
  [["K"], "previous"],
  [["M"], "toggleRead"],
  [["S"], "toggleStar"],
  [["V"], "openOriginal"],
  [["R"], "refresh"],
  [["/"], "search"],
  [["Shift", "A"], "markAllRead"],
  [["Esc"], "close"],
  [["Ctrl", "B"], "toggleSidebar"],
  [["?"], "help"],
]

export function ShortcutsDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("shortcuts.title")}</DialogTitle>
        </DialogHeader>
        <ul className="divide-y">
          {shortcuts.map(([keys, label]) => (
            <li
              key={label}
              className="flex items-center justify-between py-2 text-sm"
            >
              <span>{t(`shortcuts.${label}`)}</span>
              <span className="flex gap-1">
                {keys.map((k) => (
                  <Kbd key={k}>{k}</Kbd>
                ))}
              </span>
            </li>
          ))}
        </ul>
      </DialogContent>
    </Dialog>
  )
}
