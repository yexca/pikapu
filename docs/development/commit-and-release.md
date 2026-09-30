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
4. Merge the release commit to `main` and wait for CI to pass.
5. Tag the release commit with the exact `VERSION` value and push the tag.

## Continuous Integration

`.github/workflows/ci.yml` runs on pull requests and pushes to `main`. Its
jobs call the same Makefile targets as local validation: `make test-backend`,
`make test-frontend`, and `make smoke` (the smoke build uses Buildx with a
layer cache).

## Release Workflow

Pushing a `v*.*.*` tag runs `.github/workflows/release.yml`:

1. Verify that the tag equals `VERSION`, that `docs/history/<tag>.md`
   exists, that the `DOCKERHUB_TOKEN` secret is set, and that CI passed on
   `main` for the tagged commit (it waits up to 30 minutes for a running CI).
2. Build the production image without pushing, so a broken build stops the
   release before any registry tag moves.
3. Create a draft GitHub release with the release notes.
4. Push the image to Docker Hub (`<owner>/pikapu`) and the GitHub Container
   Registry (`ghcr.io/<owner>/pikapu`) as `<version>`, `<major>.<minor>`, and
   `latest` (for example `0.1.0`, `0.1`, `latest`), with provenance and an
   SBOM attached.
5. Publish the GitHub release.

Repository setup:

- Secret `DOCKERHUB_TOKEN`: a Docker Hub access token with read and write
  access for the Docker Hub account named after the repository owner.
- GitHub Container Registry uses the workflow's `GITHUB_TOKEN`; no secret is
  needed.

A failed run can be re-run from the Actions page; the draft release is
updated in place rather than duplicated.

Released database migrations are immutable; see
[Data model](../architecture/data-model.md#migrations).
