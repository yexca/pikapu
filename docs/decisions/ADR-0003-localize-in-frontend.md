# ADR-0003: Localize in the Frontend, Not the Backend

## Status

Accepted.

## Context

The UI supports English and Simplified Chinese, with English as the fallback.
Error messages originate in the backend (validation, fetch failures), and feed
failures are stored and displayed long after the request that caused them.
Language is a per-browser preference, not an instance setting.

## Decision

- The backend is language-neutral: API errors carry a short English `error`
  and a stable machine-readable `code`; feed failures store
  `last_error_code` alongside the English `last_error`.
- The frontend owns all user-visible text through i18next. It maps codes to
  `errors.<code>` in its catalogs and passes the English message as
  `{{detail}}` where useful (for example, `HTTP 404 Not Found`).
- English (`en.ts`) is the canonical catalog; other catalogs are typed against
  it so missing keys fail the build.

## Consequences

- Switching language never requires a server round trip, and stored failures
  display in the current language.
- Adding an error requires a code and translations in every catalog; unknown
  codes still render the English message.
- No `Accept-Language` negotiation is needed on the server.
