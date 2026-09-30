# ADR-0002: Pure-Go SQLite

## Status

Accepted.

## Context

A single-user reader stores a modest amount of data (feeds, articles,
settings) and needs durable, low-maintenance storage that lives in the data
directory. CGO complicates cross-compilation and small static images.

## Decision

Use SQLite through `modernc.org/sqlite`, a CGO-free driver. Open the database
in WAL mode with foreign keys, a busy timeout, and `BEGIN IMMEDIATE`
transactions. Manage the schema with an ordered, append-only migration list
tracked by `PRAGMA user_version`.

## Consequences

- `CGO_ENABLED=0` static builds and an Alpine runtime image.
- The whole state is one directory to back up.
- The deployment is single-instance by design; there is no support for
  multiple writers or replicas.
- Search uses `LIKE` rather than FTS, which is adequate at personal scale and
  handles CJK text without a tokenizer.
- The race detector still runs in tests (`make test-backend` uses a Go image
  with a C toolchain), independent of the CGO-free production build.
