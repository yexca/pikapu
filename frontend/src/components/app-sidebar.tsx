import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router"
import { toast } from "sonner"
import {
  CheckCheckIcon,
  ChevronRightIcon,
  CircleAlertIcon,
  ExternalLinkIcon,
  FolderPlusIcon,
  InboxIcon,
  MonitorIcon,
  MoonIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  Settings2Icon,
  StarIcon,
  SunIcon,
  Trash2Icon,
} from "lucide-react"

import { useDialogs } from "@/components/dialogs/dialogs-provider"
import { FeedIcon } from "@/components/feed-icon"
import { Logo } from "@/components/logo"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  useSidebar,
} from "@/components/ui/sidebar"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useTick } from "@/hooks/use-debounced"
import { feedErrorMessage } from "@/i18n/errors"
import { usePrefs } from "@/lib/prefs"
import {
  useCategories,
  useCounters,
  useDeleteCategory,
  useDeleteFeed,
  useFeeds,
  useMarkAllRead,
  useRefreshAll,
  useRefreshFeed,
} from "@/lib/queries"
import { relativeTime } from "@/lib/time"
import type { Category, Feed } from "@/lib/types"
import { cn } from "@/lib/utils"
import { sameView, useView, viewPath, type View } from "@/lib/view"

// Hide the unread count while the row's action button is visible.
const badgeHideOnAction =
  "md:group-hover/menu-item:opacity-0 md:group-focus-within/menu-item:opacity-0 group-has-[[data-sidebar=menu-action][data-state=open]]/menu-item:opacity-0"

function useGo() {
  const navigate = useNavigate()
  const { isMobile, setOpenMobile } = useSidebar()
  return (view: View) => {
    navigate(viewPath(view))
    if (isMobile) setOpenMobile(false)
  }
}

export function AppSidebar() {
  const { t } = useTranslation()
  const feedsQuery = useFeeds()
  const { data: categories = [] } = useCategories()
  const { data: counters } = useCounters()
  const view = useView()
  const go = useGo()
  const dialogs = useDialogs()
  const [prefs, setPrefs] = usePrefs()

  const feeds = useMemo(() => feedsQuery.data ?? [], [feedsQuery.data])
  const { byCategory, loose } = useMemo(() => {
    const byCategory = new Map<number, Feed[]>()
    const loose: Feed[] = []
    for (const f of feeds) {
      if (f.category_id == null) loose.push(f)
      else
        byCategory.set(f.category_id, [
          ...(byCategory.get(f.category_id) ?? []),
          f,
        ])
    }
    return { byCategory, loose }
  }, [feeds])

  const unread = (id: number) => counters?.feeds[String(id)] ?? 0
  const toggleCollapsed = (id: number) => {
    const set = new Set(prefs.collapsed)
    if (set.has(id)) set.delete(id)
    else set.add(id)
    setPrefs({ collapsed: [...set] })
  }

  return (
    <Sidebar>
      <SidebarHeader className="px-3 pt-3 pb-1">
        <div className="flex items-center gap-2 px-1">
          <Logo className="size-6" />
          <span className="text-[15px] font-semibold tracking-tight">
            Pikapu
          </span>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon-sm"
                variant="ghost"
                className="ml-auto"
                onClick={dialogs.addFeed}
                aria-label={t("nav.addFeed")}
              >
                <PlusIcon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{t("nav.addFeed")}</TooltipContent>
          </Tooltip>
        </div>
      </SidebarHeader>

      <SidebarContent className="scroll-thin">
        <SidebarGroup>
          <SidebarMenu>
            <NavItem
              icon={<InboxIcon />}
              label={t("nav.all")}
              count={counters?.unread}
              active={view.kind === "all"}
              onClick={() => go({ kind: "all" })}
            />
            <NavItem
              icon={<StarIcon />}
              label={t("nav.starred")}
              count={counters?.starred}
              active={view.kind === "starred"}
              onClick={() => go({ kind: "starred" })}
            />
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>{t("nav.feeds")}</SidebarGroupLabel>
          <Tooltip>
            <TooltipTrigger asChild>
              <SidebarGroupAction
                onClick={() => dialogs.category()}
                aria-label={t("nav.newCategory")}
              >
                <FolderPlusIcon />
              </SidebarGroupAction>
            </TooltipTrigger>
            <TooltipContent side="right">{t("nav.newCategory")}</TooltipContent>
          </Tooltip>
          <SidebarGroupContent>
            <SidebarMenu>
              {feedsQuery.isPending &&
                Array.from({ length: 5 }, (_, i) => (
                  <SidebarMenuItem key={i}>
                    <SidebarMenuSkeleton showIcon />
                  </SidebarMenuItem>
                ))}
              {categories.map((c) => {
                const list = byCategory.get(c.id) ?? []
                const collapsed = prefs.collapsed.includes(c.id)
                return (
                  <CategoryItem
                    key={c.id}
                    category={c}
                    feedCount={list.length}
                    unread={list.reduce((n, f) => n + unread(f.id), 0)}
                    active={sameView(view, { kind: "category", id: c.id })}
                    collapsed={collapsed}
                    onToggle={() => toggleCollapsed(c.id)}
                    onSelect={() => go({ kind: "category", id: c.id })}
                  >
                    {!collapsed &&
                      list.map((f) => (
                        <FeedItem
                          key={f.id}
                          feed={f}
                          unread={unread(f.id)}
                          active={sameView(view, { kind: "feed", id: f.id })}
                          onSelect={() => go({ kind: "feed", id: f.id })}
                        />
                      ))}
                  </CategoryItem>
                )
              })}
              {loose.map((f) => (
                <FeedItem
                  key={f.id}
                  feed={f}
                  unread={unread(f.id)}
                  active={sameView(view, { kind: "feed", id: f.id })}
                  onSelect={() => go({ kind: "feed", id: f.id })}
                />
              ))}
            </SidebarMenu>
            {feedsQuery.isSuccess && feeds.length === 0 && (
              <div className="mx-2 mt-2 rounded-lg border border-dashed px-3 py-5 text-center">
                <p className="text-xs text-muted-foreground">
                  {t("nav.noFeeds")}
                </p>
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-3"
                  onClick={dialogs.addFeed}
                >
                  <PlusIcon />
                  {t("nav.addFirstFeed")}
                </Button>
              </div>
            )}
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border/60 pt-2 pb-[max(--spacing(2),env(safe-area-inset-bottom))]">
        <FooterBar feeds={feeds} refreshing={!!counters?.refreshing} />
      </SidebarFooter>
    </Sidebar>
  )
}

function NavItem({
  icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: React.ReactNode
  label: string
  count?: number
  active: boolean
  onClick: () => void
}) {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} onClick={onClick}>
        {icon}
        <span>{label}</span>
      </SidebarMenuButton>
      {!!count && (
        <SidebarMenuBadge className="text-muted-foreground">
          {count}
        </SidebarMenuBadge>
      )}
    </SidebarMenuItem>
  )
}

function CategoryItem({
  category,
  feedCount,
  unread,
  active,
  collapsed,
  onToggle,
  onSelect,
  children,
}: {
  category: Category
  feedCount: number
  unread: number
  active: boolean
  collapsed: boolean
  onToggle: () => void
  onSelect: () => void
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const dialogs = useDialogs()
  const markAll = useMarkAllRead()
  const remove = useDeleteCategory()
  const navigate = useNavigate()

  const onDelete = async () => {
    const ok = await dialogs.confirm({
      title: t("category.deleteTitle", { name: category.name }),
      description:
        feedCount > 0
          ? t("category.deleteDescription", { count: feedCount })
          : undefined,
      confirmText: t("common.delete"),
      destructive: true,
    })
    if (!ok) return
    remove.mutate(category.id, {
      onSuccess: () => {
        if (active) navigate("/")
        toast.success(t("category.deleted"))
      },
    })
  }

  return (
    <>
      <SidebarMenuItem>
        <button
          type="button"
          onClick={onToggle}
          aria-label={collapsed ? t("nav.expand") : t("nav.collapse")}
          aria-expanded={!collapsed}
          className="absolute top-1.5 left-1.5 z-10 flex size-5 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-sidebar-border hover:text-sidebar-foreground"
        >
          <ChevronRightIcon
            className={cn(
              "size-3.5 transition-transform duration-200",
              !collapsed && "rotate-90"
            )}
          />
        </button>
        <SidebarMenuButton
          isActive={active}
          onClick={onSelect}
          className="pl-8 font-medium"
        >
          <span>{category.name}</span>
        </SidebarMenuButton>
        {unread > 0 && (
          <SidebarMenuBadge
            className={cn("text-muted-foreground", badgeHideOnAction)}
          >
            {unread}
          </SidebarMenuBadge>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuAction showOnHover className="max-md:hidden">
              <MoreHorizontalIcon />
              <span className="sr-only">{t("common.more")}</span>
            </SidebarMenuAction>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="right" align="start" className="w-44">
            <DropdownMenuItem
              onClick={() => markAll.mutate({ categoryId: category.id })}
            >
              <CheckCheckIcon />
              {t("category.markAllRead")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => dialogs.category(category)}>
              <PencilIcon />
              {t("category.rename")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={onDelete}>
              <Trash2Icon />
              {t("category.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
      {!collapsed && feedCount > 0 && (
        <li className="animate-in duration-150 fade-in-0 slide-in-from-top-1">
          <SidebarMenu className="pl-3">{children}</SidebarMenu>
        </li>
      )}
    </>
  )
}

export function FeedItem({
  feed,
  unread,
  active,
  onSelect,
}: {
  feed: Feed
  unread: number
  active: boolean
  onSelect: () => void
}) {
  const { t } = useTranslation()
  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} onClick={onSelect}>
        <FeedIcon feed={feed} />
        <span className={cn(!unread && !active && "text-muted-foreground")}>
          {feed.title}
        </span>
      </SidebarMenuButton>
      {feed.error_count > 0 ? (
        <SidebarMenuBadge className={badgeHideOnAction}>
          <Tooltip>
            <TooltipTrigger asChild>
              <CircleAlertIcon className="pointer-events-auto size-3.5 text-destructive/70" />
            </TooltipTrigger>
            <TooltipContent side="right" className="max-w-64">
              {t("feed.updateFailed", { reason: feedErrorMessage(feed) })}
            </TooltipContent>
          </Tooltip>
        </SidebarMenuBadge>
      ) : (
        unread > 0 && (
          <SidebarMenuBadge
            className={cn("text-muted-foreground", badgeHideOnAction)}
          >
            {unread}
          </SidebarMenuBadge>
        )
      )}
      <FeedMenu feed={feed} active={active}>
        <SidebarMenuAction showOnHover className="max-md:hidden">
          <MoreHorizontalIcon />
          <span className="sr-only">{t("common.more")}</span>
        </SidebarMenuAction>
      </FeedMenu>
    </SidebarMenuItem>
  )
}

/** Actions for a single feed, shared by the sidebar and the list header. */
export function FeedMenu({
  feed,
  active,
  children,
}: {
  feed: Feed
  active: boolean
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const dialogs = useDialogs()
  const refresh = useRefreshFeed()
  const markAll = useMarkAllRead()
  const remove = useDeleteFeed()
  const navigate = useNavigate()

  const onDelete = async () => {
    const ok = await dialogs.confirm({
      title: t("feed.unsubscribeTitle", { title: feed.title }),
      description: t("feed.unsubscribeDescription"),
      confirmText: t("feed.unsubscribe"),
      destructive: true,
    })
    if (!ok) return
    remove.mutate(feed.id, {
      onSuccess: () => {
        if (active) navigate("/")
        toast.success(t("feed.unsubscribed"))
      },
    })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>{children}</DropdownMenuTrigger>
      <DropdownMenuContent side="right" align="start" className="w-44">
        <DropdownMenuItem
          disabled={refresh.isPending}
          onClick={() => refresh.mutate(feed.id)}
        >
          <RefreshCwIcon />
          {t("feed.refresh")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => markAll.mutate({ feedId: feed.id })}>
          <CheckCheckIcon />
          {t("feed.markAllRead")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => dialogs.editFeed(feed)}>
          <PencilIcon />
          {t("feed.edit")}
        </DropdownMenuItem>
        {feed.site_url && (
          <DropdownMenuItem asChild>
            <a href={feed.site_url} target="_blank" rel="noreferrer">
              <ExternalLinkIcon />
              {t("feed.visitSite")}
            </a>
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onClick={onDelete}>
          <Trash2Icon />
          {t("feed.unsubscribe")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function FooterBar({
  feeds,
  refreshing,
}: {
  feeds: Feed[]
  refreshing: boolean
}) {
  useTick(30_000)
  const { t } = useTranslation()
  const dialogs = useDialogs()
  const refreshAll = useRefreshAll()
  const { theme, setTheme } = useTheme()

  const lastFetched = feeds.reduce<string | null>(
    (max, f) =>
      f.last_fetched_at && (!max || f.last_fetched_at > max)
        ? f.last_fetched_at
        : max,
    null
  )
  const busy = refreshing || refreshAll.isPending

  return (
    <div className="flex items-center gap-0.5">
      <span className="min-w-0 flex-1 truncate pl-2 text-xs text-muted-foreground">
        {busy
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
            disabled={busy}
            onClick={() => refreshAll.mutate()}
            aria-label={t("nav.refreshAll")}
          >
            <RefreshCwIcon className={cn(busy && "animate-spin")} />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">{t("nav.refreshAll")}</TooltipContent>
      </Tooltip>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button size="icon-sm" variant="ghost" aria-label={t("nav.theme")}>
            <SunIcon className="dark:hidden" />
            <MoonIcon className="hidden dark:block" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="top" align="end" className="w-36">
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
        <TooltipContent side="top">{t("nav.settings")}</TooltipContent>
      </Tooltip>
    </div>
  )
}
