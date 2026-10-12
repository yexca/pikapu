# Local Development

## Prerequisites

- Docker with Compose (required: tests run in containers)
- GNU Make
- Go 1.26+ and Node.js 22+ for running the app outside Docker

On Windows, Git Bash with GNU Make works; the Makefile disables MSYS path
rewriting for Docker arguments.

## Run the App

Two terminals, from the repository root:

```sh
make backend-run      # Go server on :7660, data in ./data
make frontend-dev     # Vite dev server on :5173, proxies /api to :7660
```

Open <http://localhost:5173>. Frontend changes hot-reload; restart
`backend-run` after Go changes.

`make backend-run` starts the server in development mode, which turns
sign-in off so previews open straight into the reader. To work on the setup
and sign-in pages, run it in production mode and use the setup token from its
output:

```sh
PIKAPU_MODE=production make backend-run
```

The Docker image and both Compose files always default to production mode.

## Run the Image

To run this checkout the way it ships, build it into the production image
with the development stack:

```sh
make docker-up        # builds pikapu:dev and serves it on :7660
make docker-logs      # setup token and recent log lines
make docker-down
```

These wrap `deploy/compose/dev.yml`. It uses `./data` and the same Compose
project as the root `docker-compose.yml`, which pulls the published image
instead of building, so `make docker-up` replaces an instance started from
that file and reads the same database. See
[Development Stack](../operations/docker.md#development-stack).

## Validate

```sh
make test             # Docker: backend, frontend, then smoke test of the image
```

See [Testing](testing.md) for the individual targets.

## Useful Paths

| Path | Contents |
| --- | --- |
| `backend/cmd/pikapu` | Entry point |
| `backend/internal/api` | HTTP handlers and routes |
| `backend/internal/fetcher` | Fetching, parsing, sanitizing |
| `backend/internal/store` | SQLite queries and migrations |
| `frontend/src/components` | UI components |
| `frontend/src/i18n/locales` | Translation catalogs |
| `docker-compose.yml` | Production stack: runs the published image |
| `deploy/compose/dev.yml` | Development stack: builds this checkout |
| `scripts/smoke.mjs` | Container smoke test |
| `docs/` | Documentation |

## Adding shadcn/ui Components

```sh
cd frontend
npx shadcn@latest add <component>
```

Generated files go to `src/components/ui`. Replace any user-visible strings
they contain with translations (see [Internationalization](i18n.md)).

## Related Docs

- [Testing](testing.md)
- [Internationalization](i18n.md)
- [Architecture](../architecture/index.md)
- [Commit and release](commit-and-release.md)
