import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { AuthCard } from "@/components/auth-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage } from "@/i18n/errors"
import { MIN_PASSWORD_LENGTH } from "@/lib/account"
import { api } from "@/lib/api"
import { markSignedIn } from "@/lib/queries"

/** First-run page that creates the admin account with the logged token. */
export function SetupPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [token, setToken] = useState("")
  const [username, setUsername] = useState("admin")
  const [password, setPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const setup = useMutation({
    mutationFn: () =>
      api.setup({ token: token.trim(), username: username.trim(), password }),
    onSuccess: (res) => markSignedIn(qc, res.username),
  })

  const tooShort = password.length > 0 && password.length < MIN_PASSWORD_LENGTH
  const mismatch = confirm.length > 0 && confirm !== password
  const ready =
    !!token.trim() &&
    !!username.trim() &&
    password.length >= MIN_PASSWORD_LENGTH &&
    confirm === password

  return (
    <AuthCard
      subtitle={t("setup.subtitle")}
      onSubmit={() => ready && !setup.isPending && setup.mutate()}
    >
      <div className="grid gap-2">
        <Label htmlFor="setup-token">{t("setup.token")}</Label>
        <Input
          id="setup-token"
          autoFocus
          autoComplete="off"
          autoCapitalize="none"
          spellCheck={false}
          className="font-mono"
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          {t("setup.tokenHint")}{" "}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-[11px]">
            docker compose logs pikapu
          </code>
        </p>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="setup-username">{t("login.username")}</Label>
        <Input
          id="setup-username"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          maxLength={64}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="setup-password">{t("login.password")}</Label>
        <Input
          id="setup-password"
          type="password"
          autoComplete="new-password"
          maxLength={128}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
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
      <div className="grid gap-2">
        <Label htmlFor="setup-confirm">{t("account.confirmPassword")}</Label>
        <Input
          id="setup-confirm"
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
      {setup.isError && (
        <p className="text-xs text-destructive">{errorMessage(setup.error)}</p>
      )}
      <Button type="submit" disabled={!ready || setup.isPending}>
        {setup.isPending && <Spinner />}
        {t("setup.submit")}
      </Button>
    </AuthCard>
  )
}
