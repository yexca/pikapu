# Agent Guide

Read these first:

- `README.md`
- `docs/overview.md`
- `docs/architecture/index.md`
- `docs/architecture/backend.md`
- `docs/architecture/frontend.md`
- `docs/architecture/data-model.md`
- `docs/operations/docker.md`
- `docs/development/testing.md`
- `docs/development/i18n.md`

## Product Boundaries

Pikapu is a single-user, self-hosted RSS reader shipped as one container.

- One Go binary serves the JSON API under `/api` and the embedded React build
  everywhere else. Do not introduce a separate frontend server or a second
  runtime process.
- SQLite is the only datastore, and the deployment is single-instance. Do not
  add features that assume multiple writers, users, or replicas.
- Access control is one admin account with server-side sessions (ADR-0004).
  Every `/api` route except `healthz` and `auth/*` must stay behind the auth
  middleware. `PIKAPU_MODE` defaults to `production`; `development` turns
  sign-in off for local previews only. Keep production the default everywhere
  an image or Compose file is involved, and do not add other ways to skip
  sign-in.
- Per-browser preferences (theme, language, text size, list filters) live in
  `localStorage`. Server-wide settings (refresh interval, retention) live in the
  `settings` table.

## Deployment Boundary

Pikapu is installed by putting `docker-compose.yml` in an empty directory and
running `docker compose up -d`. Details are in `docs/operations/docker.md`.

- The root `docker-compose.yml` is the production stack. It runs the published
  image and must keep working with nothing beside it: no checkout, no build,
  no `.env`, and no variable without a default. Do not add `build:`, a second
  service, or a mount of a repository file to it.
- `deploy/compose/dev.yml` is the development stack. It builds the checkout as
  `pikapu:dev` and otherwise matches the production stack; the `make docker-*`
  targets run it with the repository root as the project directory. Change
  the environment list, isolation, port, or mount in both files together.
- A new environment variable is added to `internal/config`, both Compose
  files, `.env.example`, and the tables in `docs/operations/configuration.md`
  and `docs/operations/docker.md`.
- The container runs with a read-only root filesystem and no Linux
  capabilities except `DAC_OVERRIDE`. Durable state goes under `/data` and
  scratch files under `/tmp`; do not write anywhere else or depend on another
  capability without an explicit decision.
- `./data` in a checkout can hold a real instance's database, and both stacks
  started from the repository root share one Compose project. Do not start,
  stop, or recreate that stack, and do not modify or delete `data/`, to test a
  change. Use `make smoke`, which runs a disposable container, or a separate
  project name, port, and directory.

## Untrusted Content Boundary

Feed documents, their HTML, linked images, favicons, and redirects are
untrusted input.

- Article HTML is rendered with `dangerouslySetInnerHTML` only after the
  backend sanitizes it (`internal/fetcher/content.go`, bluemonday policy).
  Never render feed HTML that did not pass through `SanitizeHTML`, and do not
  loosen the policy (styles, scripts, event handlers, arbitrary iframes)
  without an explicit decision.
- Embedded iframes are limited to the `iframeRe` allowlist; unknown embeds are
  turned into links. The page's Content Security Policy takes its frame
  origins from `fetcher.EmbedOrigins`, so change both together.
- Keep outbound fetches bounded: request timeouts, `maxFeedSize`,
  `maxPageSize`, `maxIconSize`, and the refresh concurrency limit. A new
  outbound path must set equivalent limits.
- Fetch errors are reduced to a stable code plus a short English message
  (`fetcher.Classify`). Never return raw transport errors containing URLs or
  credentials from the API.

## Internationalization

- The UI supports English (`en`) and Simplified Chinese (`zh-Hans`); English is
  canonical and the fallback for any unsupported browser language.
- Every user-visible string goes through i18next (`useTranslation` in
  components, `i18n.t` outside them). Add each key to
  `frontend/src/i18n/locales/en.ts` and `zh-Hans.ts`; the `Messages` type makes
  a missing or extra key a compile error.
- The backend never returns localized text. API errors are
  `{"error": "<English message>", "code": "<stable_code>"}`; add a matching
  `errors.<code>` entry to both locales for every new code.
- Dates and relative times use `Intl` with the active locale (`lib/time.ts`).

## Code Organization

- Backend: `cmd/pikapu` composes `internal/config` (environment),
  `internal/api` (HTTP), `internal/service` (adding feeds, refresh
  scheduling, OPML import and export), `internal/filter` (keyword filter
  matching), `internal/recommend` (ranking hub picks), `internal/auth`
  (password hashing, sign-in rate limiting, account bootstrap),
  `internal/fetcher` (network, parsing, sanitizing), `internal/opml` (OPML
  parsing and writing), and `internal/store` (SQLite). Dependencies point
  downward; `store` and `fetcher` must not import `api`, `service`, `filter`,
  `recommend`, or `auth`.
- Frontend: components under `src/components`, generated shadcn/ui primitives
  under `src/components/ui`, data access in `src/lib` (API client, TanStack
  Query hooks), translations in `src/i18n`. Prefer adding shadcn components
  with the shadcn CLI over hand-writing primitives.
- Schema changes append a new entry to `migrations` in
  `backend/internal/store/store.go`. Released migrations are immutable; never
  edit or reorder an existing entry.

## Testing: Docker by Default

Docker is the default test environment. The application ships as a Linux
container, so validation runs in containers rather than on the host
toolchain. The `Makefile` is the canonical entry point; prefer its targets
over reconstructing commands by hand.

- `make test-backend` runs the gofmt check, `go vet`, and race-enabled Go
  tests in a `golang` container.
- `make test-frontend` runs `npm ci`, the Prettier check, ESLint, the
  TypeScript build, and a production Vite build in a `node` container. It does
  not touch the host `node_modules` or `dist`.
- `make smoke` validates both Compose files, builds the production image, and
  exercises a disposable container, under the production stack's isolation,
  through the public HTTP contract (auth, SPA fallback, validation codes, OPML
  import/export, fetch-failure codes). It needs no outbound network access.
- `make test` runs all three and is the required check before handing back a
  change to code, the image, or a Compose file.

Keep validation proportional while iterating: `make test-backend` for backend
changes, `make test-frontend` for frontend changes, `make smoke` for Docker,
Compose, runtime, or API contract changes. A documentation-only change needs
no test run; check that its relative links resolve. Local targets such as
`make backend-test` or `make frontend-lint` are fine for quick feedback but do
not replace the Docker targets. Do not report work as verified based on
host-only runs.

Tests should protect a user-visible behavior, an API contract, a security
boundary, or a prior regression. Use reserved `.invalid` / `.test` / `example.com`
domains and obviously synthetic values in fixtures; never real feed data,
credentials, or databases.

## Documentation

Docs describe the current system, not how it got there. Where each topic
belongs is in [Documentation Rules](docs/README.md#documentation-rules).

- Repository documentation is written in English. `docs/readme/` holds
  translated READMEs; update them in the same change when `README.md` changes
  its features, quick start, or configuration.
- Update the matching page under `docs/` whenever behavior, configuration, or
  the API contract changes. Rewrite the affected section as if it had always
  been this way instead of appending a paragraph, and search for every
  statement the change contradicts: `README.md`, user and operations pages,
  `.env.example`, Makefile help text, and code comments.
- Write in present tense. Do not narrate change ("previously", "no longer",
  "now", "was removed") or cite versions, commits, or pull requests outside
  `docs/history/` and the context of an ADR.
- State each fact in one place and link to it instead of restating it.
- Add a line to `docs/history/unreleased.md` for every user-facing change,
  with any step an operator must take when upgrading at the top; see
  [Release Notes](docs/development/commit-and-release.md#release-notes).

## Release and Handoff

- `VERSION` is the single source of the application version. Read it instead
  of hard-coding the current release.
- The production stack defaults to the `latest` image tag, so a release
  reaches existing instances at their next pull. A change that an existing
  `data/` directory or `.env` cannot absorb unattended needs a migration or
  an upgrade note.
- Commit messages use `<type>(scope): <description>`, for example
  `feat(reader): add text size setting`.
- Never commit `data/`, `.env`, SQLite databases, logs, or personal feed lists.
