import { useTranslation } from "react-i18next"
import { MonitorSmartphoneIcon, XIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"
import { errorMessage } from "@/i18n/errors"
import { describeUserAgent } from "@/lib/account"
import {
  useRevokeOtherSessions,
  useRevokeSession,
  useSessions,
} from "@/lib/queries"
import { fullTime, relativeTime } from "@/lib/time"

export function SessionsDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const sessions = useSessions(open)
  const revoke = useRevokeSession()
  const revokeOthers = useRevokeOtherSessions()
  const others = sessions.data?.filter((s) => !s.current).length ?? 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("sessions.title")}</DialogTitle>
          <DialogDescription>{t("sessions.description")}</DialogDescription>
        </DialogHeader>

        {sessions.isPending ? (
          <div className="flex justify-center py-6">
            <Spinner />
          </div>
        ) : sessions.isError ? (
          <p className="text-sm text-destructive">
            {errorMessage(sessions.error)}
          </p>
        ) : (
          <ul className="divide-y">
            {sessions.data.map((s) => (
              <li key={s.id} className="flex items-center gap-3 py-3">
                <MonitorSmartphoneIcon className="size-4 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div
                    className="truncate text-sm font-medium"
                    title={s.user_agent}
                  >
                    {describeUserAgent(s.user_agent) ||
                      t("sessions.unknownDevice")}
                  </div>
                  <p
                    className="truncate text-xs text-muted-foreground"
                    title={t("sessions.signedIn", {
                      time: fullTime(s.created_at),
                    })}
                  >
                    {s.current
                      ? t("sessions.thisDevice")
                      : t("sessions.lastActive", {
                          time: relativeTime(s.last_seen_at),
                        })}
                    {s.ip && ` · ${s.ip}`}
                  </p>
                </div>
                {!s.current && (
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("sessions.revoke")}
                    title={t("sessions.revoke")}
                    disabled={revoke.isPending}
                    onClick={() => revoke.mutate(s.id)}
                  >
                    <XIcon />
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            disabled={others === 0 || revokeOthers.isPending}
            onClick={() => revokeOthers.mutate()}
          >
            {revokeOthers.isPending && <Spinner />}
            {t("sessions.revokeOthers")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
