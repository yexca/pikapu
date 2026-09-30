# Configuration

Pikapu is configured by environment variables for process-level options and
by the Settings dialog for everything else.

## Environment Variables

These are read by the `pikapu` binary. The Docker image sets the address and
data directory; Compose passes the password and time zone from `.env` (see
[Docker](docker.md#configure-with-env)).

| Variable | Binary default | Image default | Description |
| --- | --- | --- | --- |
| `PIKAPU_ADDR` | `:7660` | `:7660` | Listen address. |
| `PIKAPU_DATA_DIR` | `./data` | `/data` | Directory for `pikapu.db`. Created if missing. |
| `PIKAPU_PASSWORD` | empty | empty | When set, every API route except `/api/healthz` and `/api/auth/*` requires signing in with this password. |
| `TZ` | system | `UTC` (Compose) | Time zone for log output. Stored timestamps are always UTC. |

## Instance Settings

Changed in **Settings → Feed updates** and stored in the database:

| Setting | Default | Accepted values |
| --- | --- | --- |
| Refresh interval | 30 minutes | 5–1440 minutes (the UI offers 15 minutes to 24 hours) |
| Keep read articles | 90 days | 0–3650 days; 0 keeps everything |

Browser-only preferences (language, theme, text size, auto mark-as-read,
unread filter, collapsed categories) are stored in `localStorage` and are not
part of the instance configuration. See [Settings](../user/settings.md).

## Fixed Limits

| Limit | Value |
| --- | --- |
| Concurrent feed fetches | 6 |
| Feed request timeout | 30 seconds (60 seconds per refresh attempt overall) |
| Maximum feed size | 20 MB |
| Maximum page size for feed and icon discovery | 4 MB |
| Maximum favicon size | 512 KB |
| Failure backoff | 2×, 4×, 8×, up to 16× the refresh interval |
| Retention cleanup | about every 12 hours |
| Session lifetime | 30 days |
