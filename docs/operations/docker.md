# Docker

Pikapu ships as one image that contains the Go server and the built frontend.
The Compose file in the repository builds that image locally.

## Start

```sh
docker compose up -d --build
docker compose logs pikapu | grep setup_token
```

Open the app and enter the setup token to create the admin account, or set
`PIKAPU_ADMIN_PASSWORD` in `.env` before the first start to skip this step.

- Web app and API: <http://localhost:7660>
- Data: `./data` on the host, mounted at `/data` (the database is
  `/data/pikapu.db`)
- The container restarts automatically (`restart: unless-stopped`) and reports
  health through `pikapu healthcheck`, which calls `/api/healthz`.

`make docker-up`, `make docker-down`, `make docker-status`, and
`make docker-logs` wrap the same Compose commands.

## Prebuilt Images

Each release publishes the image to Docker Hub and the GitHub Container
Registry:

- `yexca/pikapu`
- `ghcr.io/yexca/pikapu`

Tags are the version without the `v` prefix (`0.1.0`), the minor line
(`0.1`), and `latest`. To run a published image instead of building
locally, set `PIKAPU_IMAGE` in `.env` and start without `--build`:

```sh
PIKAPU_IMAGE=yexca/pikapu:0.1
```

```sh
docker compose pull
docker compose up -d
```

## Configure with `.env`

Copy [`.env.example`](../../.env.example) to `.env` beside
`docker-compose.yml` and set only what you need. Compose reads it
automatically; exported shell variables take precedence.

| Variable | Compose default | Purpose |
| --- | --- | --- |
| `PIKAPU_IMAGE` | `pikapu:latest` | Tag for the locally built image, or a published image to pull |
| `PIKAPU_PORT` | `7660` | Host port mapped to the container's port 7660 |
| `PIKAPU_MODE` | `production` | `development` turns sign-in off; local use only |
| `PIKAPU_ADMIN_USERNAME` | `admin` | Username for the bootstrap account |
| `PIKAPU_ADMIN_PASSWORD` | empty | Creates the admin account on first start |
| `PIKAPU_TRUSTED_PROXIES` | empty | Reverse proxy addresses allowed to set `X-Forwarded-For` |
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
forward all paths to port 7660. Forward:

- `Host`, unchanged, so cross-origin protection can compare it with `Origin`;
- `X-Forwarded-Proto: https` when the proxy terminates TLS, so the session
  cookie is `Secure`;
- `X-Forwarded-For`, and list the proxy's address in
  `PIKAPU_TRUSTED_PROXIES`, so sign-in rate limiting sees client addresses.

```nginx
location / {
    proxy_pass http://127.0.0.1:7660;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

With the proxy on the host and the container's port mapped as above, the
proxy reaches Pikapu through Docker's bridge gateway; trust it with, for
example, `PIKAPU_TRUSTED_PROXIES=172.16.0.0/12`. With the proxy in another
container on the same network, trust that network's range.

## Reset the Password

```sh
docker exec pikapu pikapu reset-password
```

It prints the username and a new random password and signs out every
device. Change the password in Settings afterwards.

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
