Planned as `v0.1.0`, the first release.

## Reading

- Three-pane reader with a single-column layout on narrow screens.
- All articles, Starred, category, and feed views with an Unread/All filter.
- Search within the current view.
- Keyboard shortcuts (`J`/`K`, `M`, `S`, `V`, `R`, `/`, `Shift+A`, `?`).
- Light and dark themes and adjustable article text size.
- Hub layout: a message-center style stream with "For you" picks ranked by
  freshness, feed affinity (articles read or starred), and feed rarity, and
  updates grouped by day and source; the reader opens in a side panel.
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
- Admin account for internet-facing instances: username and Argon2id-hashed
  password, created with a one-time setup token from the log or with
  `PIKAPU_ADMIN_USERNAME` / `PIKAPU_ADMIN_PASSWORD`; editable in Settings.
  Replaces `PIKAPU_PASSWORD`, which is no longer read.
- Server-side sessions: a device list in Settings, signing out one or all
  other devices, and signing out other devices when the password changes.
- Sign-in rate limiting per client and overall, with
  `PIKAPU_TRUSTED_PROXIES` for reverse proxies.
- Cross-origin request protection and a Content Security Policy for the app.
- `pikapu reset-password` for a forgotten password.
- `PIKAPU_MODE`: `production` (default) requires sign-in; `development`
  turns it off for local previews and is the default for
  `make backend-run`.
- Health check subcommand and `/api/healthz` with the version.

## Development

- Docker-based validation: `make test-backend`, `make test-frontend`,
  `make smoke`, `make test`.
- `VERSION` as the single version source.
- GitHub Actions CI runs the containerized backend, frontend, and smoke
  checks; tagged releases publish the image to Docker Hub and the GitHub
  Container Registry and create a GitHub release.
- API errors use `{error, code}` with stable codes.
