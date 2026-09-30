import { useEffect, useRef } from "react"
import { Navigate, Route, Routes, useLocation } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { AppSidebar } from "@/components/app-sidebar"
import { DialogsProvider } from "@/components/dialogs/dialogs-provider"
import { EntriesView } from "@/components/entries-view"
import { LoginPage } from "@/components/login-page"
import { Logo } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { errorMessage } from "@/i18n/errors"
import { api, setUnauthorizedHandler } from "@/lib/api"
import { consumeUserRefresh, keys, useCounters } from "@/lib/queries"
import type { AuthStatus } from "@/lib/types"

export default function App() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const auth = useQuery({
    queryKey: keys.auth,
    queryFn: api.authStatus,
    staleTime: Infinity,
    retry: 1,
  })

  useEffect(() => {
    setUnauthorizedHandler(() =>
      qc.setQueryData<AuthStatus>(keys.auth, (s) =>
        s ? { ...s, authenticated: false } : s
      )
    )
  }, [qc])

  if (auth.isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Logo className="size-10 animate-pulse" />
      </div>
    )
  }
  if (auth.isError) {
    return (
      <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-6 text-center">
        <Logo className="size-10 grayscale" />
        <div>
          <p className="text-sm font-medium">{t("app.loadFailed")}</p>
          <p className="mt-1 text-sm text-muted-foreground">
            {errorMessage(auth.error)}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => auth.refetch()}>
          {t("app.retry")}
        </Button>
      </div>
    )
  }
  if (auth.data.auth_required && !auth.data.authenticated) {
    return <LoginPage />
  }
  return <Shell />
}

function Shell() {
  useRefreshWatcher()
  useUnreadBadge()
  const location = useLocation()

  return (
    <DialogsProvider>
      <SidebarProvider className="h-svh overflow-hidden">
        <AppSidebar />
        <SidebarInset className="min-h-0 overflow-hidden">
          <Routes>
            {["/", "/starred", "/feeds/:id", "/categories/:id"].map((path) => (
              <Route
                key={path}
                path={path}
                // Remount per view so search text and scroll position reset.
                element={<EntriesView key={location.pathname} />}
              />
            ))}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </SidebarInset>
      </SidebarProvider>
    </DialogsProvider>
  )
}

/**
 * Shows the unread count in the page title and, for an installed app, on
 * its icon where the platform supports app badges.
 */
function useUnreadBadge() {
  const unread = useCounters().data?.unread ?? 0

  useEffect(() => {
    document.title = unread > 0 ? `(${unread}) Pikapu` : "Pikapu"
    // Rejects or is missing where badges are unsupported or not permitted.
    const badge =
      unread > 0 ? navigator.setAppBadge?.(unread) : navigator.clearAppBadge?.()
    badge?.catch(() => {})
  }, [unread])

  useEffect(
    () => () => {
      document.title = "Pikapu"
      navigator.clearAppBadge?.().catch(() => {})
    },
    []
  )
}

/** Reloads data when a background refresh finishes. */
function useRefreshWatcher() {
  const qc = useQueryClient()
  const refreshing = !!useCounters().data?.refreshing
  const was = useRef(refreshing)

  useEffect(() => {
    if (was.current && !refreshing) {
      qc.invalidateQueries({ queryKey: keys.feeds })
      if (consumeUserRefresh()) {
        qc.invalidateQueries({ queryKey: keys.entryLists })
      }
    }
    was.current = refreshing
  }, [refreshing, qc])
}
