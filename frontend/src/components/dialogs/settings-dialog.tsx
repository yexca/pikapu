import { useRef } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  Columns3Icon,
  DownloadIcon,
  InboxIcon,
  ListFilterIcon,
  LogOutIcon,
  MonitorIcon,
  MoonIcon,
  SunIcon,
  UploadIcon,
} from "lucide-react"

import { useTheme, type Theme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { LOCALES, type LocalePreference } from "@/i18n"
import { api, opmlExportUrl } from "@/lib/api"
import { usePrefs, type Prefs } from "@/lib/prefs"
import {
  keys,
  useImportOpml,
  useSaveSettings,
  useSettings,
} from "@/lib/queries"
import type { AuthStatus } from "@/lib/types"

import { useDialogs } from "./dialogs-provider"

const intervals = [15, 30, 60, 120, 360, 720, 1440]
const retentions = [30, 60, 90, 180, 365, 0]

function Row({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <div className="min-w-0">
        <div className="text-sm font-medium">{title}</div>
        {description && (
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

function Section({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section>
      <h3 className="mb-1 text-xs font-medium tracking-wide text-muted-foreground">
        {title}
      </h3>
      <div className="divide-y">{children}</div>
    </section>
  )
}

/** A Select over numeric options that tolerates values not in the list. */
function NumberSelect({
  value,
  options,
  label,
  onChange,
  disabled,
}: {
  value: number | undefined
  options: readonly number[]
  label: (v: number) => string
  onChange: (v: number) => void
  disabled?: boolean
}) {
  const all =
    value !== undefined && !options.includes(value)
      ? [value, ...options]
      : options
  return (
    <Select
      value={value === undefined ? undefined : String(value)}
      onValueChange={(v) => onChange(Number(v))}
      disabled={disabled}
    >
      <SelectTrigger size="sm" className="w-32">
        <SelectValue placeholder="…" />
      </SelectTrigger>
      <SelectContent align="end">
        {all.map((v) => (
          <SelectItem key={v} value={String(v)}>
            {label(v)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function SettingsDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("settings.title")}</DialogTitle>
        </DialogHeader>
        <SettingsBody />
      </DialogContent>
    </Dialog>
  )
}

function SettingsBody() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { theme, setTheme } = useTheme()
  const [prefs, setPrefs] = usePrefs()
  const settings = useSettings()
  const save = useSaveSettings()
  const importOpml = useImportOpml()
  const dialogs = useDialogs()
  const fileRef = useRef<HTMLInputElement>(null)
  const auth = qc.getQueryData<AuthStatus>(keys.auth)

  const saveSetting = (patch: Partial<NonNullable<typeof settings.data>>) => {
    if (settings.data) save.mutate({ ...settings.data, ...patch })
  }

  const intervalLabel = (m: number) =>
    m % 60 === 0
      ? t("settings.hours", { count: m / 60 })
      : t("settings.minutes", { count: m })
  const retentionLabel = (d: number) =>
    d === 0
      ? t("settings.forever")
      : d === 365
        ? t("settings.oneYear")
        : t("settings.days", { count: d })

  const logout = async () => {
    await api.logout().catch(() => {})
    qc.setQueryData<AuthStatus>(keys.auth, (s) =>
      s ? { ...s, authenticated: false } : s
    )
    qc.removeQueries({
      predicate: (q) => q.queryKey[0] !== keys.auth[0],
    })
  }

  return (
    <div className="grid gap-6">
      <Section title={t("settings.appearance")}>
        <Row title={t("settings.language")}>
          <Select
            value={prefs.language}
            onValueChange={(v) => setPrefs({ language: v as LocalePreference })}
          >
            <SelectTrigger size="sm" className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end">
              <SelectItem value="auto">{t("settings.languageAuto")}</SelectItem>
              {LOCALES.map((l) => (
                <SelectItem key={l.value} value={l.value} lang={l.value}>
                  {l.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Row>
        <Row
          title={t("settings.layout")}
          description={
            prefs.layout === "hub"
              ? t("settings.layoutHubHint")
              : t("settings.layoutClassicHint")
          }
        >
          <ToggleGroup
            type="single"
            variant="outline"
            size="sm"
            value={prefs.layout}
            onValueChange={(v) =>
              v && setPrefs({ layout: v as Prefs["layout"] })
            }
          >
            <ToggleGroupItem value="classic" className="px-2.5">
              <Columns3Icon />
              {t("settings.layoutClassic")}
            </ToggleGroupItem>
            <ToggleGroupItem value="hub" className="px-2.5">
              <InboxIcon />
              {t("settings.layoutHub")}
            </ToggleGroupItem>
          </ToggleGroup>
        </Row>
        <Row title={t("settings.theme")}>
          <ToggleGroup
            type="single"
            variant="outline"
            size="sm"
            value={theme}
            onValueChange={(v) => v && setTheme(v as Theme)}
          >
            <ToggleGroupItem value="light" aria-label={t("theme.light")}>
              <SunIcon />
            </ToggleGroupItem>
            <ToggleGroupItem value="dark" aria-label={t("theme.dark")}>
              <MoonIcon />
            </ToggleGroupItem>
            <ToggleGroupItem value="system" aria-label={t("theme.system")}>
              <MonitorIcon />
            </ToggleGroupItem>
          </ToggleGroup>
        </Row>
        <Row title={t("settings.fontSize")}>
          <ToggleGroup
            type="single"
            variant="outline"
            size="sm"
            value={prefs.fontSize}
            onValueChange={(v) =>
              v && setPrefs({ fontSize: v as Prefs["fontSize"] })
            }
          >
            <ToggleGroupItem value="sm" className="px-3">
              {t("settings.fontSmall")}
            </ToggleGroupItem>
            <ToggleGroupItem value="base" className="px-3">
              {t("settings.fontMedium")}
            </ToggleGroupItem>
            <ToggleGroupItem value="lg" className="px-3">
              {t("settings.fontLarge")}
            </ToggleGroupItem>
          </ToggleGroup>
        </Row>
      </Section>

      <Section title={t("settings.reading")}>
        <Row title={t("settings.autoMarkRead")}>
          <Switch
            checked={prefs.autoMarkRead}
            onCheckedChange={(v) => setPrefs({ autoMarkRead: v })}
          />
        </Row>
        <Row
          title={t("settings.filters")}
          description={t("settings.filtersHint")}
        >
          <Button variant="outline" size="sm" onClick={() => dialogs.filters()}>
            <ListFilterIcon />
            {t("settings.manageFilters")}
          </Button>
        </Row>
      </Section>

      <Section title={t("settings.updates")}>
        <Row
          title={t("settings.refreshInterval")}
          description={t("settings.refreshIntervalHint")}
        >
          <NumberSelect
            value={settings.data?.refresh_interval_minutes}
            options={intervals}
            label={intervalLabel}
            disabled={!settings.data}
            onChange={(v) => saveSetting({ refresh_interval_minutes: v })}
          />
        </Row>
        <Row
          title={t("settings.retention")}
          description={t("settings.retentionHint")}
        >
          <NumberSelect
            value={settings.data?.retention_days}
            options={retentions}
            label={retentionLabel}
            disabled={!settings.data}
            onChange={(v) => saveSetting({ retention_days: v })}
          />
        </Row>
      </Section>

      <Section title={t("settings.data")}>
        <Row title={t("settings.opml")} description={t("settings.opmlHint")}>
          <div className="flex gap-2">
            <input
              ref={fileRef}
              type="file"
              accept=".opml,.xml,text/xml,application/xml"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) importOpml.mutate(file)
                e.target.value = ""
              }}
            />
            <Button
              variant="outline"
              size="sm"
              disabled={importOpml.isPending}
              onClick={() => fileRef.current?.click()}
            >
              {importOpml.isPending ? <Spinner /> : <UploadIcon />}
              {t("settings.import")}
            </Button>
            <Button variant="outline" size="sm" asChild>
              <a href={opmlExportUrl} download>
                <DownloadIcon />
                {t("settings.export")}
              </a>
            </Button>
          </div>
        </Row>
      </Section>

      {auth?.auth_required && (
        <Section title={t("settings.account")}>
          <Row title={t("settings.signOut")}>
            <Button variant="outline" size="sm" onClick={logout}>
              <LogOutIcon />
              {t("settings.signOut")}
            </Button>
          </Row>
        </Section>
      )}

      <p className="text-center text-xs text-muted-foreground">
        {t("settings.about", { version: __APP_VERSION__ })}
      </p>
    </div>
  )
}
