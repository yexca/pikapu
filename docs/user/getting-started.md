# Getting Started

## Install

Pikapu runs with Docker Compose. From a checkout of the repository:

```sh
docker compose up -d --build
```

Open <http://localhost:7660>. The first screen is an empty reader with a
sidebar on the left. If the instance was started with `PIKAPU_PASSWORD`, sign
in first. See [Docker](../operations/docker.md) for deployment details.

## Add Your First Feed

1. Click **+** at the top of the sidebar.
2. Paste a feed URL (`https://example.com/feed.xml`) or just a website address
   (`example.com`). Pikapu looks for the site's feed automatically.
3. Optionally choose a category, or pick **New category…** to create one.
4. Click **Subscribe**. Pikapu fetches the feed and opens it.

Already using another reader? Export an OPML file there and import it under
**Settings → Data → OPML → Import**.

## Read

- Pick **All articles**, **Starred**, a category, or a single feed in the
  sidebar.
- Click an article in the list to read it on the right. Opening an article
  marks it as read (you can turn this off in Settings).
- Press `J` / `K` to move between articles and `?` to see every shortcut.

More in [Reading](reading.md).

## Choose a Language

Pikapu follows your browser's language when it is English or Simplified
Chinese and falls back to English otherwise. Change it under
**Settings → Appearance → Language**. The choice is stored in this browser
only.

## Next Steps

- [Reading](reading.md)
- [Subscriptions](subscriptions.md)
- [Settings](settings.md)
