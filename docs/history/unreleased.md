Planned as `v0.1.0`, the first release.

## Reading

- Three-pane reader with a single-column layout on narrow screens.
- All articles, Starred, category, and feed views with an Unread/All filter.
- Search within the current view.
- Keyboard shortcuts (`J`/`K`, `M`, `S`, `V`, `R`, `/`, `Shift+A`, `?`).
- Light and dark themes and adjustable article text size.
- Installable as an app on phones and desktops (web app manifest, home-screen
  icons), with the unread count in the page title and on the app icon.

## Subscriptions

- Add feeds by feed URL or website address with automatic discovery.
- Categories, renaming, and unsubscribing.
- Favicon discovery and caching.
- OPML import and export.
- Localized reasons for failed feed updates.
- Keyword filters for all feeds or one feed: mark new articles as read or
  skip them, match titles or content, and optionally apply to current unread
  articles.

## Settings and Languages

- English and Simplified Chinese UI; follows the browser language and falls
  back to English, or can be chosen in Settings.
- Configurable refresh interval and read-article retention.

## Operations

- Single Docker image serving the app and API on port 7660.
- Optional password protection with signed session cookies.
- Health check subcommand and `/api/healthz` with the version.

## Development

- Docker-based validation: `make test-backend`, `make test-frontend`,
  `make smoke`, `make test`.
- `VERSION` as the single version source.
- API errors use `{error, code}` with stable codes.
