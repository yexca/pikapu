# Backend

Go module `pikapu` in `backend/`. Router: chi. Database driver:
`modernc.org/sqlite`. Feed parsing: gofeed. HTML: goquery and bluemonday.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/pikapu` | Composition root: config, account bootstrap, store, service, HTTP server, graceful shutdown, `healthcheck` and `reset-password` subcommands |
| `internal/config` | Environment variables |
| `internal/auth` | Admin account rules without HTTP: Argon2id password hashing, username and password validation, the sign-in rate limiter, creating the account from the environment, resetting the password |
| `internal/buildinfo` | Version injected at link time from `VERSION` |
| `internal/api` | Routes, JSON helpers, error contract, sessions and account endpoints, cross-origin protection, SPA file serving and its CSP |
| `internal/service` | Adding feeds, refresh orchestration and scheduling, retention cleanup, favicon lookup, OPML import/export |
| `internal/fetcher` | HTTP fetching, feed discovery, item conversion, HTML sanitizing, favicon discovery, error classification |
| `internal/filter` | Keyword filter matching: compiles `store.Filter` rows into a triage function for new entries |
| `internal/recommend` | Ranks recent unread entries for the hub layout's "For you" picks |
| `internal/store` | SQLite access, migrations, queries |
| `internal/opml` | OPML parsing and writing |
| `web` | `go:embed` of the frontend build (`web/dist`) |

Dependencies point downward: `api → service → filter → fetcher → store`,
`api → recommend → store`, and `api → auth → store`. `store` and `fetcher`
never import `api`, `service`, `filter`, `recommend`, or `auth`.

## Refresh Pipeline

1. **Scheduling.** `Service.Run` wakes three seconds after start and then
   every minute. It selects due feeds: never fetched, or older than the
   refresh interval multiplied by the failure backoff (2^errors, capped at
   16×). An atomic flag ensures only one bulk refresh runs at a time; the
   counters endpoint exposes it as `refreshing`.
2. **Fetching.** Up to six feeds are fetched concurrently. Each request sends
   `If-None-Match` / `If-Modified-Since`; `304 Not Modified` only updates the
   timestamp.
3. **Parsing.** gofeed detects RSS, Atom, or JSON Feed.
4. **Conversion** (`fetcher.ConvertItems`): resolve links against the feed and
   item URLs; choose content (content, then description, Media RSS
   description, iTunes summary); sanitize; drop a leading heading that repeats
   the title; derive a plain-text summary; pick a thumbnail (item image, Media
   RSS, image enclosure, iTunes image, or first `<img>`); clamp future
   timestamps; derive a GUID when the feed lacks one; embed YouTube videos.
5. **Saving** (`store.SaveEntries`): insert unseen GUIDs and update the text of
   known ones in one transaction. After the first fetch, unseen items older
   than the retention window are skipped so cleaned-up articles do not return.
   Unseen items then pass through the feed's [filters](#filters), which store
   them as read or drop them; known entries are never re-filtered.
6. **Bookkeeping.** Success clears the error state; failure stores
   `last_error_code`, the English `last_error`, and increments `error_count`.

## Content Sanitizing

`fetcher.SanitizeHTML` first rewrites the document with goquery: it restores
lazy-loaded images (`data-src` and similar), removes 1×1 tracking pixels,
resolves relative `href`, `src`, `srcset`, and `poster` URLs, adds
`loading="lazy"`, and converts non-allowlisted iframes to links. bluemonday's
UGC policy, extended with figure, picture, video, audio, and allowlisted
iframes, then removes everything else. Plain-text descriptions are converted
to paragraphs.

## Filters

`service.save` loads the filters for the feed and for all feeds
(`store.FiltersForFeed`) and compiles them with `filter.Compile`. The
resulting `Triage` returns `Drop` if any skip rule applies, otherwise
`KeepRead` if any mark-as-read rule applies, otherwise `Keep`.

A rule applies when any keyword occurs in the text (or, with `invert`, when
none does). The text is the title, plus the article's plain text
(`fetcher.PlainText`) for `match_content` rules. Matching is case-insensitive
(`strings.ToLower`). A keyword edge that is a letter or digit of a
space-separated script must sit at a word boundary, so `ai` does not match
"said"; Han, Hiragana, Katakana, and Hangul characters never need or form a
boundary, so CJK keywords match anywhere and `ai` matches "AI绘画".

`POST /filters/{id}/apply` runs one filter through `store.TriageUnread` over
unread, unstarred entries in its scope, marking or deleting them in one
transaction.

## Recommendations

`GET /api/entries/recommended` ranks up to 1,000 unread entries published in
the last 7 days (`recommend.Window`) with `recommend.Rank`, a transparent
heuristic rather than a learned model:

- **Freshness** halves every 48 hours.
- **Affinity** is a per-feed interest score stored on the `feeds` row. Reading
  an entry individually (`PATCH` with `is_read: true`) adds 1, starring adds
  3, and the total halves every 30 days. Mark all as read and filters do not
  count. The boost saturates, at most 2.5× for feeds read very often.
- **Rarity** favors feeds that publish less: a feed posting once a week
  outranks one posting dozens of times a day.
- Entries with an image get a small boost.

Picks are chosen greedily, and each entry already picked from a feed halves
the score of that feed's remaining entries, so one busy feed cannot fill the
list. Each pick carries the most notable `reason` (see
[HTTP API](api.md#routes)).

## Feed Discovery

`fetcher.Discover` normalizes the input URL, fetches it, and parses it as a
feed. If that fails, it collects `<link rel="alternate">` feed links from the
HTML (honoring `<base href>`), then tries `/feed`, `/rss`, `/feed.xml`,
`/rss.xml`, `/atom.xml`, and `/index.xml`. The first candidate that parses
wins.

## Favicons

Icons are looked up lazily by `GET /api/feeds/{id}/icon`. The lookup ranks
`<link rel="icon">` and `apple-touch-icon` candidates by type and size, falls
back to `/favicon.ico`, validates the bytes as an image, and stores them in
the `feeds` row. `singleflight` prevents duplicate lookups; icons are
refreshed after 30 days, failed lookups retried after 3 days.

## Errors

`fetcher.Classify` maps any fetch error to a `*fetcher.Error` with a stable
code (`fetch_timeout`, `fetch_dns`, `fetch_http`, `fetch_parse`,
`fetch_too_large`, `fetch_canceled`, `fetch_network`, `feed_not_found`,
`invalid_url`) and a short English message without the URL. See
[HTTP API](api.md#errors).

## Authentication

See [ADR-0004](../decisions/ADR-0004-admin-account.md) for the reasoning.

- **Modes.** `PIKAPU_MODE=production` (the default) requires sign-in for
  every `/api` route except `healthz` and `auth/*`. `development` lets every
  request through; `auth/status` reports `mode: "development"` so the UI
  skips the sign-in page.
- **Account.** One row in `account` holds the username and a PHC-encoded
  Argon2id hash (19 MiB, 2 passes, 1 lane; parameters are read back from the
  hash). Usernames compare case-insensitively. The account is never cached in
  memory, so `pikapu reset-password` in a second process takes effect at once.
- **First run.** At startup, `auth.EnsureAccount` creates the account from
  `PIKAPU_ADMIN_USERNAME` / `PIKAPU_ADMIN_PASSWORD` if none exists. Otherwise
  `api.New` generates a random setup token, logs it as `setup_token=…`, and
  keeps it in memory; `POST /auth/setup` requires it and works once.
- **Sessions.** Signing in stores the SHA-256 of a random 32-byte token in
  `sessions` and sets the token as an `HttpOnly`, `SameSite=Lax` cookie
  (`__Host-pikapu_session` over HTTPS, `pikapu_session` otherwise). Each
  request looks the token up; activity is written back at most hourly, which
  also extends the 30-day idle expiry. Signing out deletes the row, and a
  password change deletes every other session in the same transaction.
- **Rate limiting.** `auth.Limiter` counts failed setup, sign-in, and
  current-password checks per client IP: five free failures, then waits of
  30 s doubling to 15 min, forgotten after an hour without failures. A global
  budget of 50 failures refilling one per 6 s caps guessing across addresses.
  Throttled requests get `429 too_many_attempts` with `Retry-After`. The
  client IP is the socket peer unless that peer is in
  `PIKAPU_TRUSTED_PROXIES`, in which case the nearest untrusted
  `X-Forwarded-For` hop is used.
- **Cross-origin requests.** `http.CrossOriginProtection` rejects
  state-changing requests that browsers mark as cross-origin
  (`Sec-Fetch-Site`, or an `Origin` that does not match `Host`) with
  `403 cross_origin`. Non-browser clients that send neither header pass.
- **Content Security Policy.** `index.html` is served with a CSP that allows
  scripts only from the app's origin plus the SHA-256 of each inline script
  in the built page, frames only from `fetcher.EmbedOrigins`, and no plugins.
  Images and media may load from anywhere, as feed articles need.
