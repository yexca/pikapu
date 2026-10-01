import { useTranslation } from "react-i18next"
import {
  Columns3Icon,
  InboxIcon,
  MonitorIcon,
  MoonIcon,
  RefreshCwIcon,
  Settings2Icon,
  SunIcon,
} from "lucide-react"

import { useDialogs } from "@/components/dialogs/dialogs-provider"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { SidebarTrigger } from "@/components/ui/sidebar"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useTick } from "@/hooks/use-debounced"
import { usePrefs } from "@/lib/prefs"
import { useFeeds } from "@/lib/queries"
import { relativeTime } from "@/lib/time"
import { cn } from "@/lib/utils"

export interface AppHeaderProps {
  title: string
  subtitle?: string
  refreshing: boolean
  onRefresh: () => void
  className?: string
}

/**
 * The top bar shared by both layouts: the current view on the left, and
 * refresh, layout, theme, and settings on the right.
 */
export function AppHeader({
  title,
  subtitle,
  refreshing,
  onRefresh,
  className,
}: AppHeaderProps) {
  useTick(30_000)
  const { t } = useTranslation()
  const dialogs = useDialogs()
  const { data: feeds = [] } = useFeeds()

  const lastFetched = feeds.reduce<string | null>(
    (max, f) =>
      f.last_fetched_at && (!max || f.last_fetched_at > max)
        ? f.last_fetched_at
        : max,
    null
  )

  return (
    <header
      className={cn(
        "flex h-12 shrink-0 items-center gap-1 border-b bg-background px-3",
        className
      )}
    >
      <SidebarTrigger className="-ml-0.5 text-muted-foreground" />
      <div className="min-w-0 flex-1 px-1">
        <h1 className="truncate text-[15px] leading-tight font-semibold">
          {title}
        </h1>
        {subtitle && (
          <p className="truncate text-xs text-muted-foreground">{subtitle}</p>
        )}
      </div>
      <span className="shrink-0 pr-1 text-xs text-muted-foreground max-md:hidden">
        {refreshing
          ? t("nav.refreshing")
          : lastFetched
            ? t("nav.updated", { time: relativeTime(lastFetched) })
            : ""}
      </span>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon-sm"
            variant="ghost"
            disabled={refreshing}
            onClick={onRefresh}
            aria-label={t("list.refresh")}
          >
            <RefreshCwIcon className={cn(refreshing && "animate-spin")} />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("list.refreshHint")}</TooltipContent>
      </Tooltip>
      <LayoutSwitch />
      <ThemeMenu />
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon-sm"
            variant="ghost"
            onClick={dialogs.settings}
            aria-label={t("nav.settings")}
          >
            <Settings2Icon />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("nav.settings")}</TooltipContent>
      </Tooltip>
    </header>
  )
}

/** Switches between the classic and hub layouts; the icon shows the target. */
function LayoutSwitch() {
  const { t } = useTranslation()
  const [prefs, setPrefs] = usePrefs()
  const hub = prefs.layout === "hub"
  const label = hub ? t("list.switchToClassic") : t("list.switchToHub")
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          size="icon-sm"
          variant="ghost"
          onClick={() => setPrefs({ layout: hub ? "classic" : "hub" })}
          aria-label={label}
        >
          {hub ? <Columns3Icon /> : <InboxIcon />}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

function ThemeMenu() {
  const { t } = useTranslation()
  const { theme, setTheme } = useTheme()
  return (
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger asChild>
          <DropdownMenuTrigger asChild>
            <Button size="icon-sm" variant="ghost" aria-label={t("nav.theme")}>
              <SunIcon className="dark:hidden" />
              <MoonIcon className="hidden dark:block" />
            </Button>
          </DropdownMenuTrigger>
        </TooltipTrigger>
        <TooltipContent>{t("nav.theme")}</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(v) => setTheme(v as typeof theme)}
        >
          <DropdownMenuRadioItem value="light">
            <SunIcon />
            {t("theme.light")}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark">
            <MoonIcon />
            {t("theme.dark")}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="system">
            <MonitorIcon />
            {t("theme.system")}
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
