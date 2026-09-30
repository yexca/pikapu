# Agent Guide

Read these first:

- `README.md`
- `docs/overview.md`
- `docs/architecture/index.md`
- `docs/architecture/backend.md`
- `docs/architecture/frontend.md`
- `docs/architecture/data-model.md`
- `docs/development/testing.md`
- `docs/development/i18n.md`

## Product Boundaries

Pikapu is a single-user, self-hosted RSS reader shipped as one container.

- One Go binary serves the JSON API under `/api` and the embedded React build
  everywhere else. Do not introduce a separate frontend server or a second
  runtime process.
- SQLite is the only datastore, and the deployment is single-instance. Do not
  add features that assume multiple writers, users, or replicas.
- Access control is one optional password (`PIKAPU_PASSWORD`). Every `/api`
  route except `healthz` and `auth/*` must stay behind the auth middleware.
- Per-browser preferences (theme, language, text size, list filters) live in
  `localStorage`. Server-wide settings (refresh interval, retention) live in the
  `settings` table.

## Untrusted Content Boundary

Feed documents, their HTML, linked images, favicons, and redirects are
untrusted input.

- Article HTML is rendered with `dangerouslySetInnerHTML` only after the
  backend sanitizes it (`internal/fetcher/content.go`, bluemonday policy).
  Never render feed HTML that did not pass through `SanitizeHTML`, and do not
  loosen the policy (styles, scripts, event handlers, arbitrary iframes)
  without an explicit decision.
- Embedded iframes are limited to the `iframeRe` allowlist; unknown embeds are
  turned into links.
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

- Backend: `cmd/pikapu` composes `internal/api` (HTTP), `internal/service`
  (refresh scheduling, OPML), `internal/fetcher` (network, parsing,
  sanitizing), and `internal/store` (SQLite). Dependencies point downward;
  `store` and `fetcher` must not import `api` or `service`.
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
- `make smoke` builds the production image and exercises a disposable
  container through the public HTTP contract (auth, SPA fallback, validation
  codes, OPML import/export, fetch-failure codes). It needs no outbound
  network access.
- `make test` runs all three and is the required check before handing work
  back.

Keep validation proportional while iterating: `make test-backend` for backend
changes, `make test-frontend` for frontend changes, `make smoke` for Docker,
runtime, or API contract changes. Local targets such as `make backend-test`
or `make frontend-lint` are fine for quick feedback but do not replace the
Docker targets. Do not report work as verified based on host-only runs.

Tests should protect a user-visible behavior, an API contract, a security
boundary, or a prior regression. Use reserved `.invalid` / `.test` / `example.com`
domains and obviously synthetic values in fixtures; never real feed data,
credentials, or databases.

## Documentation and Handoff

- Repository documentation is written in English. `docs/readme/` holds
  translated READMEs; keep them in step with `README.md` when the quick start
  or configuration changes.
- Update the matching page under `docs/` whenever behavior, configuration, or
  the API contract changes, and add a line to `docs/history/unreleased.md`.
- `VERSION` is the single source of the application version.
- Commit messages use `<type>(scope): <description>`, for example
  `feat(reader): add text size setting`.
- Never commit `data/`, `.env`, SQLite databases, logs, or personal feed lists.
