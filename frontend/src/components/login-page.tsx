import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { Logo } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage } from "@/i18n/errors"
import { api } from "@/lib/api"
import { keys } from "@/lib/queries"
import type { AuthStatus } from "@/lib/types"

export function LoginPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [password, setPassword] = useState("")
  const login = useMutation({
    mutationFn: () => api.login(password),
    onSuccess: () => {
      qc.setQueryData<AuthStatus>(keys.auth, (s) =>
        s ? { ...s, authenticated: true } : s
      )
      qc.invalidateQueries({
        predicate: (q) => q.queryKey[0] !== keys.auth[0],
      })
    },
  })

  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault()
          if (password) login.mutate()
        }}
        className="w-full max-w-xs rounded-2xl border bg-background p-6 shadow-sm"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <Logo className="size-11" />
          <div>
            <h1 className="text-lg font-semibold tracking-tight">Pikapu</h1>
            <p className="text-xs text-muted-foreground">
              {t("login.subtitle")}
            </p>
          </div>
        </div>
        <div className="mt-6 grid gap-3">
          <Input
            type="password"
            autoFocus
            autoComplete="current-password"
            placeholder={t("login.password")}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={login.isError || undefined}
          />
          {login.isError && (
            <p className="text-xs text-destructive">
              {errorMessage(login.error)}
            </p>
          )}
          <Button type="submit" disabled={!password || login.isPending}>
            {login.isPending && <Spinner />}
            {t("login.submit")}
          </Button>
        </div>
      </form>
    </div>
  )
}
