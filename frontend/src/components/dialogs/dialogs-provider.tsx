/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import type { Category, Feed } from "@/lib/types"

import { AddFeedDialog } from "./add-feed-dialog"
import { CategoryDialog } from "./category-dialog"
import { EditFeedDialog } from "./edit-feed-dialog"
import { SettingsDialog } from "./settings-dialog"
import { ShortcutsDialog } from "./shortcuts-dialog"

interface ConfirmOptions {
  title: string
  description?: string
  confirmText?: string
  destructive?: boolean
}

interface DialogsApi {
  addFeed: () => void
  editFeed: (feed: Feed) => void
  category: (category?: Category) => void
  settings: () => void
  shortcuts: () => void
  confirm: (options: ConfirmOptions) => Promise<boolean>
}

const DialogsContext = createContext<DialogsApi | null>(null)

// Dialog payloads are kept after closing so content doesn't vanish mid-animation.
interface Slot<T> {
  open: boolean
  value: T
}

export function DialogsProvider({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation()
  const [addOpen, setAddOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [edit, setEdit] = useState<Slot<Feed | null>>({
    open: false,
    value: null,
  })
  const [category, setCategory] = useState<Slot<Category | null>>({
    open: false,
    value: null,
  })
  const [confirm, setConfirm] = useState<Slot<ConfirmOptions>>({
    open: false,
    value: { title: "" },
  })
  const resolveRef = useRef<((ok: boolean) => void) | null>(null)

  const api = useMemo<DialogsApi>(
    () => ({
      addFeed: () => setAddOpen(true),
      editFeed: (feed) => setEdit({ open: true, value: feed }),
      category: (c) => setCategory({ open: true, value: c ?? null }),
      settings: () => setSettingsOpen(true),
      shortcuts: () => setShortcutsOpen(true),
      confirm: (options) =>
        new Promise<boolean>((resolve) => {
          resolveRef.current?.(false)
          resolveRef.current = resolve
          setConfirm({ open: true, value: options })
        }),
    }),
    []
  )

  const settle = (ok: boolean) => {
    resolveRef.current?.(ok)
    resolveRef.current = null
    setConfirm((s) => ({ ...s, open: false }))
  }

  return (
    <DialogsContext.Provider value={api}>
      {children}
      <AddFeedDialog open={addOpen} onOpenChange={setAddOpen} />
      <EditFeedDialog
        feed={edit.value}
        open={edit.open}
        onOpenChange={(open) => setEdit((s) => ({ ...s, open }))}
      />
      <CategoryDialog
        category={category.value}
        open={category.open}
        onOpenChange={(open) => setCategory((s) => ({ ...s, open }))}
      />
      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <ShortcutsDialog open={shortcutsOpen} onOpenChange={setShortcutsOpen} />
      <AlertDialog
        open={confirm.open}
        onOpenChange={(open) => !open && settle(false)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirm.value.title}</AlertDialogTitle>
            {confirm.value.description && (
              <AlertDialogDescription>
                {confirm.value.description}
              </AlertDialogDescription>
            )}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => settle(false)}>
              {t("common.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              variant={confirm.value.destructive ? "destructive" : "default"}
              onClick={() => settle(true)}
            >
              {confirm.value.confirmText ?? t("common.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DialogsContext.Provider>
  )
}

export function useDialogs(): DialogsApi {
  const ctx = useContext(DialogsContext)
  if (!ctx) throw new Error("useDialogs must be used within DialogsProvider")
  return ctx
}
