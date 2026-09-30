# Backend

Go module `pikapu` in `backend/`. Router: chi. Database driver:
`modernc.org/sqlite`. Feed parsing: gofeed. HTML: goquery and bluemonday.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/pikapu` | Composition root: config, store, service, HTTP server, graceful shutdown, `healthcheck` subcommand |
| `internal/config` | Environment variables |
| `internal/buildinfo` | Version injected at link time from `VERSION` |
| `internal/api` | Routes, JSON helpers, error contract, password auth, SPA file serving |
| `internal/service` | Adding feeds, refresh orchestration and scheduling, retention cleanup, favicon lookup, OPML import/export |
| `internal/fetcher` | HTTP fetching, feed discovery, item conversion, HTML sanitizing, favicon discovery, error classification |
| `internal/store` | SQLite access, migrations, queries |
| `internal/opml` | OPML parsing and writing |
| `web` | `go:embed` of the frontend build (`web/dist`) |

Dependencies point downward: `api → service → fetcher/store`. `store` and
`fetcher` never import `api` or `service`.

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

`internal/api/auth.go` implements the optional password. Session cookies hold
an expiry and an HMAC-SHA256 signature over the expiry and a hash of the
password, keyed by a 32-byte secret stored in the `settings` table.
