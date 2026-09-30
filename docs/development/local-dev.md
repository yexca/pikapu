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
`backend-run` after Go changes. Set `PIKAPU_PASSWORD` in the backend's
environment to work on the sign-in flow.

To run the production build instead:

```sh
make docker-up        # builds the image and serves it on :7660
```

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
