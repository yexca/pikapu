# Docker

Pikapu ships as one image that contains the Go server and the built frontend.
The Compose file in the repository builds that image locally.

## Start

```sh
docker compose up -d --build
```

- Web app and API: <http://localhost:7660>
- Data: `./data` on the host, mounted at `/data` (the database is
  `/data/pikapu.db`)
- The container restarts automatically (`restart: unless-stopped`) and reports
  health through `pikapu healthcheck`, which calls `/api/healthz`.

`make docker-up`, `make docker-down`, `make docker-status`, and
`make docker-logs` wrap the same Compose commands.

## Configure with `.env`

Copy [`.env.example`](../../.env.example) to `.env` beside
`docker-compose.yml` and set only what you need. Compose reads it
automatically; exported shell variables take precedence.

| Variable | Compose default | Purpose |
| --- | --- | --- |
| `PIKAPU_IMAGE` | `pikapu:latest` | Tag for the locally built image |
| `PIKAPU_PORT` | `7660` | Host port mapped to the container's port 7660 |
| `PIKAPU_PASSWORD` | empty | Enables sign-in when set |
| `TZ` | `UTC` | Time zone for log timestamps |

After editing `.env`, apply it by recreating the container:

```sh
docker compose up -d
```

`docker compose restart` reuses the existing container and does not pick up
new environment values. See [Configuration](configuration.md) for the full
list of runtime variables.

## Bind to Localhost Only

The default mapping listens on every host interface. To keep the instance
private to the host, change the port line in `docker-compose.yml`:

```yaml
ports:
  - "127.0.0.1:7660:7660"
```

## Reverse Proxy

Pikapu serves everything from one origin, so a reverse proxy only needs to
forward all paths to port 7660. Forward `X-Forwarded-Proto: https` when the
proxy terminates TLS; Pikapu then marks the session cookie `Secure`.

```nginx
location / {
    proxy_pass http://127.0.0.1:7660;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

## Upgrade

```sh
cp -r data data.backup-$(date +%Y%m%d)
git pull
docker compose up -d --build
```

Database migrations run automatically on startup and only move forward.

## Back Up and Restore

The whole state is the `data/` directory. For a consistent copy, stop the
container first (`docker compose stop`), copy `data/`, then start it again.
To restore, stop the container, replace `data/`, and start it.

## Build Without Compose

```sh
make docker-build                  # builds pikapu:dev
docker run -d -p 7660:7660 -v "$PWD/data:/data" pikapu:dev
```

The image build reads `VERSION` and embeds it in both the backend
(`/api/healthz`) and the frontend (Settings footer).
