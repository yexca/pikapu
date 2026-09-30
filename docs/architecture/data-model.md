# Data Model

One SQLite database, `pikapu.db`, opened in WAL mode with foreign keys
enabled, a 5-second busy timeout, and `BEGIN IMMEDIATE` transactions.
Timestamps are Unix seconds (UTC).

## Tables

### `categories`

| Column | Notes |
| --- | --- |
| `id` | Primary key |
| `name` | Unique, case-insensitive (`COLLATE NOCASE`) |
| `position` | Sort order |
| `created_at` | |

### `feeds`

| Column | Notes |
| --- | --- |
| `id` | Primary key |
| `category_id` | Nullable; `ON DELETE SET NULL` |
| `title` | Set on creation, then only changed by the user |
| `feed_url` | Unique |
| `site_url`, `description` | Updated from the feed on each successful fetch |
| `etag`, `last_modified` | Conditional request validators |
| `last_fetched_at` | Last attempt, successful or not |
| `last_error`, `last_error_code`, `error_count` | Failure detail (English), stable code, consecutive failures |
| `icon`, `icon_mime`, `icon_checked_at` | Cached favicon |
| `affinity`, `affinity_at` | Interest score for recommendations and when it was last raised; decays with a 30-day half-life |
| `created_at` | |

### `entries`

| Column | Notes |
| --- | --- |
| `id` | Primary key |
| `feed_id` | `ON DELETE CASCADE` |
| `guid` | Unique per feed |
| `url`, `title`, `author` | |
| `summary` | Plain text, up to 200 characters |
| `content` | Sanitized HTML |
| `image_url` | Thumbnail for the list |
| `published_at` | Feed date, or first-seen time when missing |
| `created_at` | First-seen time |
| `is_read`, `is_starred` | 0 or 1 |

Indexes cover newest-first listing overall, per feed, unread, and starred,
plus `(feed_id, is_read)` for unread counts.

### `filters`

| Column | Notes |
| --- | --- |
| `id` | Primary key |
| `feed_id` | Nullable (all feeds); `ON DELETE CASCADE` |
| `keywords` | One keyword per line |
| `match_content` | 0: title only; 1: title and article text |
| `invert` | 1: apply when no keyword appears |
| `action` | `mark_read` or `skip` |
| `created_at` | |

Filters only act when an entry is first stored (see
[Backend](backend.md#filters)); skipped items are never written to `entries`.

### `settings`

Key/value pairs: `refresh_interval_minutes` and `retention_days`.

### `account`

At most one row (`id` is constrained to 1).

| Column | Notes |
| --- | --- |
| `username` | Compared case-insensitively on sign-in |
| `password_hash` | PHC string, `$argon2id$v=19$m=…,t=…,p=…$<salt>$<hash>` |
| `created_at`, `updated_at` | |

### `sessions`

| Column | Notes |
| --- | --- |
| `id` | Primary key |
| `token_hash` | SHA-256 of the cookie token; unique. The token itself is never stored |
| `user_agent` | Up to 256 bytes, for the device list |
| `ip` | Client address at sign-in, updated with activity |
| `created_at`, `last_seen_at` | `last_seen_at` is written at most hourly |
| `expires_at` | 30 days after `last_seen_at` |

Expired rows are ignored by lookups and deleted when a new session starts.

## Migrations

`backend/internal/store/store.go` holds an ordered list of SQL migrations.
`PRAGMA user_version` records how many have run, and each runs in its own
transaction on startup.

| # | Change |
| --- | --- |
| 1 | Initial schema |
| 2 | `feeds.last_error_code` |
| 3 | `filters` table |
| 4 | `feeds.affinity`, `feeds.affinity_at` |
| 5 | `account` and `sessions` tables; removes the old `session_secret` setting |

Released migrations are immutable. Add a schema change as the next entry;
never edit or reorder existing ones.

## Pagination

Entry lists are ordered by `published_at DESC, id DESC`. The cursor is
`<published_at>_<id>` of the last row, so pages stay stable while articles are
marked read or new ones arrive.

## Retention

When `retention_days > 0`, a cleanup about every 12 hours deletes entries
that are read, not starred, and published before the cutoff. To keep deleted
articles from being re-imported, fetches after a feed's first one skip unseen
items older than the cutoff.

## Search

Search uses `LIKE` over `title` and `content` with escaped wildcards. It is
case-insensitive for ASCII and works for CJK text without tokenization, which
suits personal-scale databases.
