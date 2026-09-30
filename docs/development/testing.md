# Testing

Docker is the default test environment. Pikapu ships as a Linux container, so
checks run in containers with pinned toolchains rather than on whatever the
host has installed. The `Makefile` is the canonical entry point.

## Targets

| Target | Runs in | What it checks |
| --- | --- | --- |
| `make test-backend` | `golang:1.26` | `gofmt`, `go vet`, `go test -race ./...` |
| `make test-frontend` | `node:24-alpine` | `npm ci`, Prettier, ESLint (no warnings), `tsc -b`, production Vite build |
| `make smoke` | built image | Builds `pikapu:dev` and drives a disposable container over HTTP |
| `make test` | all of the above | Required before handing off or merging |

`make ci` is an alias for `make test`.

Container runs keep caches in named volumes (`pikapu-gomod`,
`pikapu-gobuild`, `pikapu-npm`). The frontend container installs
dependencies into an anonymous volume, so the host `node_modules` and `dist`
are never modified. Remove the named volumes with
`docker volume rm pikapu-gomod pikapu-gobuild pikapu-npm` to start clean.

## Choosing a Target

| Change | Minimum check while iterating |
| --- | --- |
| Go code | `make test-backend` |
| Frontend code or translations | `make test-frontend` |
| Dockerfile, Compose, API contract, auth, startup | `make smoke` |
| Anything, before handoff | `make test` |

Local targets (`make backend-test`, `make backend-vet`, `make frontend-lint`,
`make frontend-format`) are useful for quick feedback, but results from the
host toolchain do not replace the Docker targets.

## Smoke Test

`scripts/smoke.mjs` starts the image with a synthetic password on
`127.0.0.1:17660` (override with `PIKAPU_SMOKE_PORT`) and verifies:

- health check and version
- SPA serving and client-route fallback
- the public web app manifest and its icons
- the auth boundary, a wrong password, and the session cookie
- category creation and case-insensitive uniqueness
- settings and URL validation error codes
- OPML import, the background fetch of a `.invalid` feed failing with a
  `fetch_*` code, and OPML export
- filter validation codes, creation, applying, and deletion
- recommendations behind the auth boundary
- sign-out

It needs no outbound network access, and it prints the container logs when a
check fails.

## Writing Tests

- Protect a user-visible behavior, an API contract, a security boundary, or a
  prior regression. Skip tests that only restate an implementation.
- Prefer the lowest layer that proves the behavior: pure functions (sanitizing,
  URL normalization, error classification) in `internal/fetcher`, queries and
  transitions against a temporary database in `internal/store`, the HTTP
  contract in the smoke test.
- Use synthetic data: reserved domains (`example.com`, `*.invalid`, `*.test`),
  generic titles such as `Example Feed`, and values such as
  `synthetic-password`. Never copy real feeds, databases, or credentials into
  fixtures.

## Existing Coverage

| Package | Covered behavior |
| --- | --- |
| `internal/fetcher` | HTML sanitizing (scripts, relative URLs, lazy images, tracking pixels, iframe allowlist), plain-text extraction, title cleanup, duplicate-heading removal, URL normalization, feed-link discovery, error classification |
| `internal/filter` | Keyword matching (case, word boundaries, CJK), keyword cleanup and limits, title vs. content rules, inverted rules, skip-over-read precedence |
| `internal/store` | Upsert semantics, retention skip, cursor pagination, read/star state and counters, category-scoped mark-all-read, search escaping, retention cleanup, category deletion, filter CRUD and scoping, triage of new vs. known entries, applying a filter to unread entries |
| `internal/api` | SPA fallback, cache headers, and the web app manifest content type |
| smoke test | Runtime HTTP contract of the built image |
