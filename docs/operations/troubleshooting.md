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

## The container keeps restarting

The log names the cause.

- `unable to open database file`: Pikapu cannot write to `/data`. Check that
  `./data` beside `docker-compose.yml` is a directory the container may write
  to, and keep `cap_add: DAC_OVERRIDE` in the Compose file (see
  [Container Isolation](docker.md#container-isolation)).
- `PIKAPU_MODE must be production or development` or an error naming
  `PIKAPU_TRUSTED_PROXIES` or `PIKAPU_ADMIN_PASSWORD`: fix that value in
  `.env` and run `docker compose up -d`.

## The image cannot be pulled

`PIKAPU_IMAGE` in `.env` or the shell names an image that is on no registry,
such as a locally built tag. Remove it to use `yexca/pikapu:latest`, or set a
published tag (see [Images](docker.md#images)).

## "Frontend is not built"

The Go binary was built without the frontend bundle. This only happens with
a manual `go build`; use `make docker-up`, or build the frontend and copy
`frontend/dist` to `backend/web/dist` before building the backend.

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

## "Please sign in" on other devices after changing the password

Expected: a new password signs out every other device. Sign in there with
the new password.

## Where is the setup token?

Pikapu prints it at every start while no account exists:
`docker compose logs pikapu | grep setup_token`. If you set
`PIKAPU_ADMIN_PASSWORD`, the account was created instead and there is no
token; sign in with those credentials.

## Forgot the password

Run `docker compose exec pikapu pikapu reset-password` and sign in with the
printed password.

## "Too many attempts"

Several failed sign-ins from one address make Pikapu wait before accepting
another attempt, up to 15 minutes. Wait, or restart the container to clear
the counters. If this happens to you behind a reverse proxy without anyone
guessing, set `PIKAPU_TRUSTED_PROXIES` so clients are told apart (see
[Docker](docker.md#reverse-proxy)).

## Still asked for a password after upgrading

`PIKAPU_PASSWORD` is no longer used. Use the setup token from the log to
create the admin account, or set `PIKAPU_ADMIN_PASSWORD` and recreate the
container.

## Starting over

Stop the container and delete `data/` (this erases all subscriptions and
articles), then start it again. Export OPML first if you want to keep your
subscriptions.
