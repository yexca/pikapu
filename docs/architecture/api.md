# HTTP API

All endpoints are under `/api` and exchange JSON. Timestamps are RFC 3339
strings. When a password is configured, every route except `healthz` and
`auth/*` requires the session cookie and otherwise returns `401` with code
`unauthorized`.

## Routes

| Method and path | Purpose |
| --- | --- |
| `GET /healthz` | `{"status":"ok","version":"v0.1.0"}` |
| `GET /auth/status` | `{"auth_required", "authenticated"}` |
| `POST /auth/login` | `{"password"}` → sets the session cookie |
| `POST /auth/logout` | Clears the session cookie |
| `GET /categories` | List categories |
| `POST /categories` | `{"name"}` |
| `PUT /categories/order` | `{"ids": [...]}` |
| `PUT /categories/{id}` | `{"name"}` |
| `DELETE /categories/{id}` | Feeds become uncategorized |
| `GET /feeds` | List feeds |
| `POST /feeds` | `{"url", "category_id"?, "category_name"?}` → discovers and fetches the feed |
| `POST /feeds/refresh` | Starts a background refresh of all feeds; `202 {"started"}` |
| `PUT /feeds/{id}` | `{"title", "feed_url", "category_id"?, "category_name"?}` |
| `DELETE /feeds/{id}` | Unsubscribe and delete its articles |
| `POST /feeds/{id}/refresh` | Refresh now; returns the feed (check `last_error_code`) |
| `GET /feeds/{id}/icon` | Favicon bytes, or 404 |
| `GET /entries` | Query: `feed_id`, `category_id`, `starred=1`, `unread=1`, `q`, `cursor`, `limit` (≤ 200) → `{"entries", "next_cursor"}`; list items omit `content` |
| `GET /entries/{id}` | Entry with sanitized `content` |
| `PATCH /entries/{id}` | `{"is_read"?, "is_starred"?}` |
| `POST /entries/mark-all-read` | `{"feed_id"?, "category_id"?, "starred"?, "q"?}` → `{"updated"}` |
| `GET /filters` | List filters, those for all feeds first |
| `POST /filters` | `{"feed_id"?, "keywords", "match_content"?, "invert"?, "action"}` → `201` filter |
| `PUT /filters/{id}` | Same body; replaces the filter |
| `DELETE /filters/{id}` | Delete a filter |
| `POST /filters/{id}/apply` | Run the filter over unread, unstarred entries in its scope → `{"updated"}` |
| `GET /counters` | `{"unread", "starred", "feeds": {"<id>": n}, "refreshing"}` |
| `GET /settings`, `PUT /settings` | `{"refresh_interval_minutes", "retention_days"}` |
| `GET /opml` | OPML download |
| `POST /opml` | Multipart field `file` → `{"added", "skipped"}` |

`category_name` creates the category if needed and takes precedence over
`category_id`.

A filter's `feed_id` is `null` (or omitted) for all feeds. `action` is
`mark_read` or `skip`; `invert: true` applies it when none of the keywords
appear; `match_content: true` also searches the article text. Keywords are
trimmed, whitespace-collapsed, and de-duplicated ignoring case. Matching rules
are in [Backend](backend.md#filters).

## Errors

Every error response has the same shape:

```json
{ "error": "category already exists", "code": "category_exists" }
```

- `error` is a short English message for logs and fallback display. For fetch
  failures it is the detail, such as `HTTP 404 Not Found`.
- `code` is stable and is what clients branch on and localize. The frontend
  maps it to `errors.<code>` in its locale catalogs.

| Code | Status | Meaning |
| --- | --- | --- |
| `unauthorized` | 401 | Sign-in required |
| `invalid_password` | 401 | Wrong password |
| `invalid_request` | 400 | Malformed JSON body |
| `invalid_id` | 400 | Path id is not a positive integer |
| `invalid_cursor` | 400 | Bad `cursor` parameter |
| `invalid_name` | 400 | Category name empty or longer than 64 characters |
| `invalid_title` | 400 | Feed title empty or longer than 200 characters |
| `url_required` | 400 | Feed URL missing |
| `invalid_url` | 400 | Not an HTTP(S) URL |
| `invalid_refresh_interval` | 400 | Outside 5–1440 minutes |
| `invalid_retention` | 400 | Outside 0–3650 days |
| `opml_required` | 400 | No file uploaded |
| `opml_invalid` | 400 | File is not OPML |
| `category_not_found` | 400 | Referenced category does not exist |
| `invalid_keywords` | 400 | Filter needs 1–50 keywords of up to 100 characters |
| `invalid_filter_action` | 400 | Filter action is not `mark_read` or `skip` |
| `unknown_feed` | 400 | Filter references a feed that does not exist |
| `category_exists` | 409 | Duplicate category name |
| `feed_exists` | 409 | Already subscribed |
| `not_found` | 404 | Unknown resource or endpoint |
| `conflict` | 409 | Other uniqueness conflict |
| `method_not_allowed` | 405 | Unsupported method |
| `feed_not_found` | 502 | No feed at the address |
| `fetch_timeout` | 502 | Upstream timeout |
| `fetch_dns` | 502 | DNS lookup failed |
| `fetch_http` | 502 | Upstream non-2xx status |
| `fetch_parse` | 502 | Upstream content is not a feed |
| `fetch_too_large` | 502 | Upstream response over the size limit |
| `fetch_canceled` | 502 | Request canceled |
| `fetch_network` | 502 | Other network failure |
| `internal` | 500 | Unexpected server error (details are logged) |

The `fetch_*` codes also appear in a feed's `last_error_code` after a failed
refresh.
