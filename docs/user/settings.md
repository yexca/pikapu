# Settings

Open Settings with the sliders button in the sidebar footer. Settings marked
*browser* are stored in this browser only; settings marked *server* apply to
the whole instance.

## Appearance

| Setting | Scope | Options |
| --- | --- | --- |
| Language | browser | System default, English, 简体中文 |
| Layout | browser | Classic, Hub; see [Reading](reading.md#layout) |
| Theme | browser | Light, Dark, System |
| Article text size | browser | Small, Medium, Large |

**System default** uses the first supported language in your browser's
language list. Simplified Chinese variants (`zh`, `zh-CN`, `zh-SG`,
`zh-Hans`) select Simplified Chinese; English variants select English; any
other language, including Traditional Chinese, falls back to English.

The theme can also be switched from the sun/moon button in the sidebar footer.

## Reading

| Setting | Scope | Default |
| --- | --- | --- |
| Mark articles as read when opened | browser | On |
| Filters | server | None; see [Filters](filters.md) |

## Feed Updates

| Setting | Scope | Default | Range |
| --- | --- | --- | --- |
| Refresh interval | server | 30 minutes | 15 minutes to 24 hours |
| Keep read articles | server | 90 days | 30 days to 1 year, or Forever |

Retention cleanup runs about twice a day and deletes read, unstarred articles
older than the limit. Starred and unread articles are never deleted. Articles
that were removed are not imported again if they are still in the feed.

## Data

Import or export subscriptions as OPML. See
[Subscriptions](subscriptions.md#opml).

## Account

When the instance has a password, **Sign out** ends the session in this
browser. Sessions last 30 days.
