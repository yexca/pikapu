# Overview

Pikapu is a personal RSS reader: one person, one container, one SQLite file.
It fetches the feeds you subscribe to on a schedule and gives you a calm place
to read them.

## Goals

- Make following many sites effortless: subscribe with a URL or homepage, read
  everything in one list, and keep track of what is read.
- Keep reading comfortable on desktop and phone, in light or dark mode, in
  English or Simplified Chinese.
- Stay simple to run: a single `docker compose up`, no external services, and
  data that fits in one directory.
- Treat feed content as untrusted: sanitize everything before it reaches the
  browser.

## Current Capabilities

- RSS 0.9x/1.0/2.0, Atom, and JSON Feed parsing (via gofeed), with feed
  discovery from HTML pages.
- Categories, per-feed and total unread counts, starring, search, and an
  unread/all filter.
- Background refresh with conditional requests, per-feed backoff after
  failures, and localized failure reasons in the UI.
- Retention cleanup of old read articles (starred and unread articles are
  kept).
- Favicon discovery and caching.
- OPML import and export.
- Optional single-password sign-in.
- English and Simplified Chinese UI with English fallback.

## Non-Goals

- Multiple user accounts or sharing.
- Full-text extraction of summary-only feeds (articles show what the feed
  provides; "Read original" opens the site).
- Horizontal scaling or an external database.
- A public multi-tenant service.

## Runtime Topology

```text
Browser ──HTTP──▶ pikapu (Go)
                   ├─ /api/*  JSON API ─▶ SQLite (/data/pikapu.db)
                   ├─ /*      embedded React build
                   └─ scheduler ─▶ feed sites (HTTP GET, conditional)
```

See [Architecture](architecture/index.md) for details.
