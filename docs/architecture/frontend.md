# Frontend

Vite, React 19, TypeScript, Tailwind CSS v4, shadcn/ui (Radix), TanStack
Query, React Router, and i18next. Source in `frontend/src`.

## Structure

| Path | Contents |
| --- | --- |
| `main.tsx` | Providers: theme, locale sync, query client, tooltips, router, toasts |
| `App.tsx` | Auth gate (setup page, sign-in page, or the app), app shell (sidebar + routes), refresh watcher |
| `components/app-header.tsx` | Top bar shared by both layouts: view title, last update, refresh, layout switch, theme, settings |
| `components/app-sidebar.tsx` | Navigation, categories, feeds |
| `components/entries-view.tsx` | Orchestrates the list and the reader for the current view in either layout: selection, keyboard shortcuts, auto mark-as-read |
| `components/entry-list.tsx` | Classic list; the list toolbar (search, Unread/All, view actions) and empty states shared with the hub |
| `components/hub-view.tsx` | Hub layout stream: "For you" picks and updates grouped by day and feed |
| `components/reader.tsx` | Article toolbar and rendering |
| `components/dialogs/` | Add/edit feed, category, filters, settings, shortcuts, confirm dialogs, exposed through `useDialogs()` |
| `components/ui/` | Generated shadcn/ui primitives |
| `lib/api.ts` | Typed API client and `ApiError` (status, code, message) |
| `lib/queries.ts` | TanStack Query hooks, optimistic updates, cache keys |
| `lib/view.ts` | URL ↔ view mapping and entry filters |
| `lib/hub.ts` | Groups the hub stream by day and feed |
| `lib/prefs.ts` | `localStorage`-backed preferences store |
| `lib/time.ts` | Locale-aware relative and absolute dates |
| `i18n/` | i18next setup, locale catalogs, error localization |

## Routing

The path selects the view and `?entry=<id>` the open article:

| Path | View |
| --- | --- |
| `/` | All articles |
| `/starred` | Starred |
| `/feeds/:id` | One feed |
| `/categories/:id` | One category |

`EntriesView` is keyed by path, so search text and scroll position reset when
the view changes. On phones, opening an article pushes a history entry so the
system back gesture returns to the list.

## Data and Caching

- Entry lists use cursor-based infinite queries keyed by the filter object.
- Hub picks (`usePicks`) are cached under the same `entries` key prefix and
  in the same page shape, so read/star updates, lookups, and refresh
  invalidation cover them too. They are ranked once per load and do not
  reshuffle while reading.
- Toggling read or star updates every cached copy of the entry and the
  counters optimistically, and rolls back on error.
- Read entries stay visible in the unread view until the list is refetched;
  lists do not refetch on window focus for that reason.
- `/api/counters` is polled every minute, and every 1.5 seconds while the
  server reports `refreshing`. When a refresh the user started finishes, entry
  lists are reloaded; scheduled refreshes only update counts.

## Layout

The **Layout** preference (`prefs.layout`, per browser) chooses between two
presentations of the same view, selection, and shortcuts.

Classic (default):

- ≥ 1024 px: sidebar, list (22–26 rem), and reader side by side.
- < 1024 px: list or reader, one at a time.
- < 768 px: the sidebar becomes a drawer.

Hub, a message-center style stream:

- A centered column (max 48 rem). In All articles and category views, not
  while searching, "For you" shows recommended picks first: a hero card, up
  to four cards, then a compact list.
- "Latest updates" groups the remaining entries by local day and, except in
  single-feed views, by feed, like notifications grouped by app. Stacks show
  three entries and expand on demand or when keyboard navigation reaches a
  hidden entry.
- The reader opens in a right-hand sheet (full screen below 640 px). J/K
  follow display order: picks, then the stream. The sheet carries
  `data-allow-hotkeys` so `useHotkeys` keeps reading shortcuts active inside
  it; other dialogs still suspend them.

## Internationalization

All strings come from `i18n/locales/en.ts` (canonical) and `zh-Hans.ts`.
`LocaleSync` applies the language preference and sets `<html lang>`. API
errors are localized from their `code` by `errorMessage()`; feed update errors
by `feedErrorMessage()`. See [Internationalization](../development/i18n.md).

## Installable App

`public/manifest.webmanifest` makes the SPA installable (`display:
standalone`). There is no service worker and no offline cache: the reader
always talks to the server, and the backend serves `.webmanifest` files as
`application/manifest+json`.

- Icons: `favicon-32.png`, `icon-192.png`, `icon-512.png`,
  `icon-maskable-512.png` (Android adaptive icons), and `apple-touch-icon.png`
  (iOS). The PNGs are committed; `make icons` resizes them in a container
  from the 512 px masters `docs/assets/pikapu-icon.png` (rounded, with
  transparent corners) and `pikapu-icon-maskable.png` (full bleed, the face
  inside the central safe zone). The `Logo` component shows `icon-192.png`,
  so the in-app logo always matches the installed icon.
- `ThemeProvider` rewrites `<meta name="theme-color">` to match the resolved
  theme, so an explicit Light or Dark choice also colors the browser toolbar
  and the installed app's title bar.
- `useUnreadBadge` (in `App.tsx`) puts the unread count in `document.title`
  and on the app icon through the Badging API where supported.
- `index.html` sets `viewport-fit=cover`, so bottom-anchored scroll areas,
  including the sidebar, pad by `env(safe-area-inset-bottom)` to stay clear of
  the iPhone home indicator.

## Mascot

The mascot is a messenger mage who delivers feeds; the app icon is her
chibi portrait. `components/mascot.tsx` shows one of eight transparent WebP
poses from `src/assets/mascot/`:

| Pose | Where |
| --- | --- |
| `welcome` | Sign-in and setup card |
| `pointing` | Classic reader pane with no article open; she points at the list |
| `reading` | End of an open article, beside **Read original** |
| `caught-up` | Entry list with no unread articles |
| `searching` | Search with no results |
| `starred` | Starred view with no starred articles |
| `waiting` | Entry list with no articles yet, including before the first feed |
| `oops` | The app or an article failed to load |

The images are decorative (`alt=""`); the text next to them carries the
meaning. Keep new poses in the same style, around 440 px tall, and under
about 60 KB each.

## Styling

Theme tokens are CSS variables in `index.css` (shadcn "nova" preset, neutral
base) plus a warm `--brand` accent used for unread markers and the logo.
Article typography uses `@tailwindcss/typography` with overrides in the
unlayered `.article` rules.
