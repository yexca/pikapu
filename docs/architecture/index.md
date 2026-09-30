# Architecture

Pikapu is a single Go process that owns everything: the HTTP API, the
embedded frontend, the refresh scheduler, and the SQLite database.

```text
Browser (React SPA)
   │  fetch /api/*  (JSON, session cookie)
   ▼
internal/api ──────────────▶ internal/store ──▶ SQLite (WAL)
   │                               ▲
   ▼                               │
internal/service ──(schedule)──────┘
   │
   ▼
internal/fetcher ──HTTP──▶ feed sites, web pages, favicons
```

## Topics

- [Backend](backend.md): packages, refresh pipeline, content processing
- [Frontend](frontend.md): layout, state, data fetching, i18n
- [Data model](data-model.md): tables, migrations, retention
- [HTTP API](api.md): routes and the error contract

## Key Decisions

- One binary with an embedded frontend and one port
  ([ADR-0001](../decisions/ADR-0001-single-binary.md)).
- Pure-Go SQLite with WAL, no CGO
  ([ADR-0002](../decisions/ADR-0002-sqlite-pure-go.md)).
- The backend speaks English with stable error codes; the frontend localizes
  ([ADR-0003](../decisions/ADR-0003-localize-in-frontend.md)).
- One admin account with server-side sessions; sign-in is always on outside
  development mode ([ADR-0004](../decisions/ADR-0004-admin-account.md)).

## When To Read What

- Changing how feeds are fetched, parsed, or sanitized: [Backend](backend.md).
- Adding a table or column: [Data model](data-model.md).
- Adding an endpoint or error: [HTTP API](api.md) and
  [Internationalization](../development/i18n.md).
- Changing screens, state, or strings: [Frontend](frontend.md).
