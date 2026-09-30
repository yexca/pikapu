import { useState } from "react"
import { useTranslation } from "react-i18next"

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
import { MIN_PASSWORD_LENGTH } from "@/lib/account"
import { useAccount, useUpdateAccount } from "@/lib/queries"

export function AccountDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const account = useAccount()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        {account.data ? (
          <AccountForm
            // Reset the form each time the dialog opens.
            key={String(open)}
            username={account.data.username}
            onDone={() => onOpenChange(false)}
          />
        ) : (
          <div className="flex justify-center py-8">
            <Spinner />
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function AccountForm({
  username: initialUsername,
  onDone,
}: {
  username: string
  onDone: () => void
}) {
  const { t } = useTranslation()
  const update = useUpdateAccount()
  const [username, setUsername] = useState(initialUsername)
  const [newPassword, setNewPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const [current, setCurrent] = useState("")

  const tooShort =
    newPassword.length > 0 && newPassword.length < MIN_PASSWORD_LENGTH
  const mismatch = confirm.length > 0 && confirm !== newPassword
  const changed = username.trim() !== initialUsername || newPassword !== ""
  const ready =
    changed &&
    !!username.trim() &&
    !tooShort &&
    confirm === newPassword &&
    !!current

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!ready || update.isPending) return
    update.mutate(
      {
        current_password: current,
        username: username.trim(),
        new_password: newPassword || undefined,
      },
      { onSuccess: onDone }
    )
  }

  return (
    <form onSubmit={submit} className="grid gap-4">
      <DialogHeader>
        <DialogTitle>{t("account.title")}</DialogTitle>
        <DialogDescription>{t("account.description")}</DialogDescription>
      </DialogHeader>

      <div className="grid gap-2">
        <Label htmlFor="account-username">{t("login.username")}</Label>
        <Input
          id="account-username"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          maxLength={64}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="account-new-password">{t("account.newPassword")}</Label>
        <Input
          id="account-new-password"
          type="password"
          autoComplete="new-password"
          maxLength={128}
          placeholder={t("account.newPasswordPlaceholder")}
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          aria-invalid={tooShort || undefined}
        />
        <p
          className={
            tooShort
              ? "text-xs text-destructive"
              : "text-xs text-muted-foreground"
          }
        >
          {t("account.passwordHint", { count: MIN_PASSWORD_LENGTH })}
        </p>
      </div>

      {newPassword && (
        <div className="grid gap-2">
          <Label htmlFor="account-confirm">
            {t("account.confirmPassword")}
          </Label>
          <Input
            id="account-confirm"
            type="password"
            autoComplete="new-password"
            maxLength={128}
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            aria-invalid={mismatch || undefined}
          />
          {mismatch && (
            <p className="text-xs text-destructive">
              {t("account.passwordMismatch")}
            </p>
          )}
        </div>
      )}

      <div className="grid gap-2 border-t pt-4">
        <Label htmlFor="account-current">{t("account.currentPassword")}</Label>
        <Input
          id="account-current"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          aria-invalid={update.error ? true : undefined}
        />
        {newPassword && (
          <p className="text-xs text-muted-foreground">
            {t("account.signOutOthersHint")}
          </p>
        )}
      </div>

      {update.isError && (
        <p className="text-xs text-destructive">{errorMessage(update.error)}</p>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={!ready || update.isPending}>
          {update.isPending && <Spinner />}
          {t("common.save")}
        </Button>
      </DialogFooter>
    </form>
  )
}
