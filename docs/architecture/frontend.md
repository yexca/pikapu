# Frontend

Vite, React 19, TypeScript, Tailwind CSS v4, shadcn/ui (Radix), TanStack
Query, React Router, and i18next. Source in `frontend/src`.

## Structure

| Path | Contents |
| --- | --- |
| `main.tsx` | Providers: theme, locale sync, query client, tooltips, router, toasts |
| `App.tsx` | Auth gate, app shell (sidebar + routes), refresh watcher |
| `components/app-sidebar.tsx` | Navigation, categories, feeds, footer actions |
| `components/entries-view.tsx` | Orchestrates the list and the reader for the current view: selection, keyboard shortcuts, auto mark-as-read |
| `components/entry-list.tsx` | List header, search, infinite scroll, empty states |
| `components/reader.tsx` | Article toolbar and rendering |
| `components/dialogs/` | Add/edit feed, category, settings, shortcuts, confirm dialogs, exposed through `useDialogs()` |
| `components/ui/` | Generated shadcn/ui primitives |
| `lib/api.ts` | Typed API client and `ApiError` (status, code, message) |
| `lib/queries.ts` | TanStack Query hooks, optimistic updates, cache keys |
| `lib/view.ts` | URL ↔ view mapping and entry filters |
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
- Toggling read or star updates every cached copy of the entry and the
  counters optimistically, and rolls back on error.
- Read entries stay visible in the unread view until the list is refetched;
  lists do not refetch on window focus for that reason.
- `/api/counters` is polled every minute, and every 1.5 seconds while the
  server reports `refreshing`. When a refresh the user started finishes, entry
  lists are reloaded; scheduled refreshes only update counts.

## Layout

- ≥ 1024 px: sidebar, list (22–26 rem), and reader side by side.
- < 1024 px: list or reader, one at a time.
- < 768 px: the sidebar becomes a drawer.

## Internationalization

All strings come from `i18n/locales/en.ts` (canonical) and `zh-Hans.ts`.
`LocaleSync` applies the language preference and sets `<html lang>`. API
errors are localized from their `code` by `errorMessage()`; feed update errors
by `feedErrorMessage()`. See [Internationalization](../development/i18n.md).

## Styling

Theme tokens are CSS variables in `index.css` (shadcn "nova" preset, neutral
base) plus a warm `--brand` accent used for unread markers and the logo.
Article typography uses `@tailwindcss/typography` with overrides in the
unlayered `.article` rules.
