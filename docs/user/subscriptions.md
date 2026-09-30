# Subscriptions

## Adding Feeds

Click **+** in the sidebar and enter an address:

- A feed URL is used directly.
- A web page is searched for `<link rel="alternate">` feed links. If none are
  found, Pikapu tries common paths such as `/feed`, `/rss.xml`, and
  `/atom.xml`.
- `feed://` addresses and URLs without a scheme are accepted; `https://` is
  assumed.

Redirects are followed and the final feed URL is stored. Subscribing to the
same feed twice is rejected.

## Categories

- Create one with the folder button next to **Feeds**, or with
  **New category…** when adding or editing a feed.
- Category names are unique, ignoring letter case.
- Use the **⋯** menu on a category to mark it read, rename it, or delete it.
  Deleting a category moves its feeds to Uncategorized; no feeds or articles
  are deleted.
- Collapsed categories stay collapsed in this browser.

## Editing and Removing Feeds

Use the **⋯** menu on a feed (in the sidebar, or in the list header when the
feed is open) to refresh it, mark it read, edit its name, URL, or category,
visit the website, or unsubscribe. Unsubscribing deletes all of the feed's
articles, including starred ones.

## Updates

Feeds refresh in the background at the interval set in
[Settings](settings.md) (30 minutes by default). **Refresh** in the list
header refreshes the open feed, or all feeds from other views. The sidebar
footer shows when feeds were last updated.

Pikapu sends conditional requests, so unchanged feeds cost little. A feed that
fails is retried less often: 2×, 4×, 8×, and at most 16× the interval.

## Update Errors

A feed whose last update failed shows a warning icon in the sidebar. Hover it,
or open **Edit feed**, to see the reason:

| Reason | Meaning |
| --- | --- |
| The site took too long to respond | No complete response within about 30 seconds |
| Couldn't resolve the site's domain name | DNS lookup failed |
| The site responded with HTTP … | Non-2xx status such as 404 or 403 |
| The response isn't a valid feed | The URL returns HTML or broken XML |
| The feed is too large | The response exceeds 20 MB |
| Network error | Connection refused, reset, TLS failure, and similar |

The warning clears after the next successful update.

## OPML

**Settings → Data → OPML** imports and exports subscriptions:

- Import adds feeds that are not already subscribed and creates categories
  from OPML folders (nested folders use the innermost name). New feeds are
  fetched in the background right away.
- Export writes every feed, grouped by category, as `pikapu-YYYYMMDD.opml`.
