import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { AuthCard } from "@/components/auth-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage } from "@/i18n/errors"
import { api } from "@/lib/api"
import { markSignedIn } from "@/lib/queries"

export function LoginPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const login = useMutation({
    mutationFn: () => api.login(username.trim(), password),
    onSuccess: (res) => markSignedIn(qc, res.username),
  })
  const ready = !!username.trim() && !!password

  return (
    <AuthCard
      subtitle={t("login.subtitle")}
      onSubmit={() => ready && !login.isPending && login.mutate()}
    >
      <Input
        autoFocus
        autoComplete="username"
        autoCapitalize="none"
        spellCheck={false}
        placeholder={t("login.username")}
        aria-label={t("login.username")}
        value={username}
        onChange={(e) => setUsername(e.target.value)}
      />
      <Input
        type="password"
        autoComplete="current-password"
        placeholder={t("login.password")}
        aria-label={t("login.password")}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        aria-invalid={login.isError || undefined}
      />
      {login.isError && (
        <p className="text-xs text-destructive">{errorMessage(login.error)}</p>
      )}
      <Button type="submit" disabled={!ready || login.isPending}>
        {login.isPending && <Spinner />}
        {t("login.submit")}
      </Button>
    </AuthCard>
  )
}
