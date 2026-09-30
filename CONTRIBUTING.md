# Contributing

Thanks for helping improve Pikapu. Small, focused changes are the easiest to
review and keep stable.

## Before Changing Code

Read the docs that match the area you are changing:

- User-visible behavior: `docs/user/`
- Deployment and configuration: `docs/operations/`
- System design and data: `docs/architecture/`
- Development workflow, testing, and translations: `docs/development/`
- Durable design decisions: `docs/decisions/`

## Core Rules

- Keep Pikapu a single-container, single-user application backed by SQLite.
- Render feed HTML only after server-side sanitization.
- Route every user-visible string through i18next and add it to both the
  English and Simplified Chinese catalogs. The backend returns English
  messages with stable error codes, never localized text.
- Append new database migrations; never edit a released one.
- Update the relevant documentation and `docs/history/unreleased.md` when
  behavior changes.
- Do not commit runtime data, `.env`, databases, logs, or personal feed lists.

## Validation

Tests run in Docker by default. Use the smallest Makefile target that covers
your change while iterating, and run the full suite before opening a pull
request:

```sh
make test-backend    # gofmt, go vet, race-enabled Go tests (golang container)
make test-frontend   # Prettier, ESLint, TypeScript, Vite build (node container)
make smoke           # production image + HTTP contract checks
make test            # all of the above
```

Local toolchain targets (`make backend-test`, `make frontend-lint`, …) give
faster feedback but do not replace the Docker targets.

## Commit Messages

Use:

```text
<type>(scope): <description>
```

Examples:

```text
feat(reader): add text size setting
fix(fetcher): keep lazy-loaded images
docs(operations): describe password protection
```

## Security Reports

Do not disclose suspected vulnerabilities in public issues or pull requests.
Follow [SECURITY.md](SECURITY.md).
