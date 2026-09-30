# Commit and Release

## Commit Format

```text
<type>(scope): <description>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `chore`.

```text
feat(reader): add text size setting
fix(fetcher): keep lazy-loaded images
docs(i18n): describe locale fallback
```

Before committing, run `make test` and review the diff for runtime data,
`.env` files, databases, logs, or personal feed lists.

## Version Source

`VERSION` holds the application version as `v<major>.<minor>.<patch>`. It is
the single source:

- the Docker build and `make backend-build` inject it into the backend
  (`/api/healthz`, startup log);
- `vite.config.ts` reads it for the Settings footer.

## Release Notes

Record user-facing changes in `docs/history/unreleased.md` as you make them,
grouped by area (Reading, Subscriptions, Settings, Operations, Development).
When releasing:

1. Set `VERSION` to the new version.
2. Move the unreleased notes to `docs/history/<version>.md` and link it from
   `docs/history/index.md`; reset `unreleased.md`.
3. Run `make test`.
4. Tag the release commit with the exact `VERSION` value.

Released database migrations are immutable; see
[Data model](../architecture/data-model.md#migrations).
