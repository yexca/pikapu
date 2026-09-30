# Troubleshooting

Start with the logs:

```sh
docker compose logs --tail 200 pikapu
```

## The page does not load

- Check the container is healthy: `docker compose ps`.
- Check the port mapping and that nothing else uses port 7660.
- `curl http://localhost:7660/api/healthz` should return
  `{"status":"ok","version":"…"}`.

## "Frontend is not built"

The Go binary was built without the frontend bundle. This only happens with
a manual `go build`; use `docker compose up --build`, or build the frontend
and copy `frontend/dist` to `backend/web/dist` before building the backend.

## A feed shows a warning icon

Hover the icon or open **Edit feed** to see the reason (see
[Update errors](../user/subscriptions.md#update-errors)). Common causes:

- **HTTP 403 / 429**: the site blocks automated clients or rate-limits. Try a
  different feed URL from the same site or a longer refresh interval.
- **Not a valid feed**: the URL returns an HTML page. Add the website address
  instead and let Pikapu discover the feed.
- **DNS or network errors**: check that the container can reach the internet
  (`docker compose exec pikapu wget -qO- https://example.com`).

Failing feeds are retried with backoff and recover automatically.

## Images don't appear in articles

Some sites block hotlinked images. Pikapu already sends no `Referer`, which
resolves most cases; the rest require opening the original article.

## "Please sign in" after changing the password

Expected: changing `PIKAPU_PASSWORD` invalidates existing sessions. Sign in
with the new password.

## Starting over

Stop the container and delete `data/` (this erases all subscriptions and
articles), then start it again. Export OPML first if you want to keep your
subscriptions.
