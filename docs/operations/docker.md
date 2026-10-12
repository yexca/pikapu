# Docker

Pikapu ships as one image that contains the Go server and the built frontend.
Each release publishes it to Docker Hub and the GitHub Container Registry.

## Default Stack

[`docker-compose.yml`](../../docker-compose.yml) runs the published image. It
needs neither the source tree nor a local build: put the file in an empty
directory and start it there. No `.env` file or password is required.

```sh
docker compose up -d
docker compose logs pikapu | grep setup_token
```

Open the app and enter the setup token to create the admin account, or set
`PIKAPU_ADMIN_PASSWORD` in `.env` before the first start to skip this step.

- Web app and API: <http://localhost:7660>
- Data: `./data` beside the Compose file, mounted at `/data` (the database is
  `/data/pikapu.db`). Docker creates the directory on first start.
- The container restarts automatically (`restart: unless-stopped`) and reports
  health through `pikapu healthcheck`, which calls `/api/healthz`.
- Stopping the container sends `SIGTERM`; Pikapu gives open requests 10
  seconds to finish. The stack sets `stop_grace_period: 20s` so Docker waits
  for that.

## Images

- `yexca/pikapu` on Docker Hub
- `ghcr.io/yexca/pikapu` on the GitHub Container Registry

Tags are the version without the `v` prefix (`0.1.1`), the minor line
(`0.1`), and `latest`, which follows each release. The stack defaults to
`yexca/pikapu:latest`. Set `PIKAPU_IMAGE` to a version or digest when the
deployment should be reproducible:

```dotenv
PIKAPU_IMAGE=yexca/pikapu:0.1.1
```

## Container Isolation

The image runs as the container's root user. The stack drops every Linux
capability, enables `no-new-privileges`, and mounts the root filesystem
read-only with a bounded writable `/tmp`. Only `/data` is durable and
writable.

It adds back `DAC_OVERRIDE` alone. Without it the container's root user
follows ordinary permission bits, so on a Linux host a `./data` directory that
belongs to another user is not writable and Pikapu fails to open its database.

## Configure with `.env`

Copy [`.env.example`](../../.env.example) to `.env` beside
`docker-compose.yml` and set only what you need. Compose reads it
automatically; exported shell variables take precedence.

| Variable | Compose default | Purpose |
| --- | --- | --- |
| `PIKAPU_IMAGE` | `yexca/pikapu:latest` | Image to run; see [Images](#images) |
| `PIKAPU_PORT` | `7660` | Host port mapped to the container's port 7660 |
| `PIKAPU_MODE` | `production` | `development` turns sign-in off; local use only |
| `PIKAPU_ADMIN_USERNAME` | `admin` | Username for the bootstrap account |
| `PIKAPU_ADMIN_PASSWORD` | empty | Creates the admin account on first start |
| `PIKAPU_TRUSTED_PROXIES` | empty | Reverse proxy addresses allowed to set `X-Forwarded-For` |
| `TZ` | `UTC` | Time zone for log timestamps |

After editing `.env`, validate it and apply it by recreating the container:

```sh
docker compose config --quiet
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

Run this in the directory that holds `docker-compose.yml`:

```sh
docker compose exec pikapu pikapu reset-password
```

It prints the username and a new random password and signs out every
device. Change the password in Settings afterwards.

## Upgrade

Back up `data/` first (see [Back Up and Restore](#back-up-and-restore)), then
pull the configured image and recreate the container:

```sh
docker compose pull
docker compose up -d
```

For a pinned version, change `PIKAPU_IMAGE` before running these commands.
Database migrations run automatically on startup and only move forward, so
restore the backup to return to an older version.

## Back Up and Restore

The whole state is the `data/` directory. For a consistent copy, stop the
container first (`docker compose stop`), copy `data/`, then start it again.
To restore, stop the container, replace `data/`, and start it.

## Development Stack

[`deploy/compose/dev.yml`](../../deploy/compose/dev.yml) builds the image from
a checkout instead of pulling it, for trying local changes in the shipped
form. Run it from the repository root, or use `make docker-up`:

```sh
docker compose --project-directory . -f deploy/compose/dev.yml up -d --build
```

Keep `--project-directory .` in direct Compose commands. It keeps the build
context, the `.env` lookup, the project name, and the `./data` mount at the
repository root even though the file lives under `deploy/compose/`.
`make docker-up`, `make docker-down`, `make docker-status`, and
`make docker-logs` set it automatically.

The development stack tags its build `pikapu:dev` and otherwise matches the
default stack: the same service name, port, `.env` variables, isolation, and
`./data` mount. Both stacks started from the repository root share one Compose
project, so starting one replaces the other's container and keeps the data.

## Build Without Compose

```sh
make docker-build                  # builds pikapu:dev
docker run -d -p 7660:7660 -v "$PWD/data:/data" pikapu:dev
```

The image build reads `VERSION` and embeds it in both the backend
(`/api/healthz`) and the frontend (Settings footer).
