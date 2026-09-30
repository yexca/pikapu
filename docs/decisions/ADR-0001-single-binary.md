# ADR-0001: Single Binary with an Embedded Frontend

## Status

Accepted.

## Context

Pikapu is a personal, self-hosted app. Operators should be able to run it with
one command and one port, without configuring a web server, CORS, or a second
container for static files.

## Decision

The Go server embeds the production frontend build with `go:embed` and serves
it next to the API: `/api/*` is the JSON API, and every other path serves a
static asset or falls back to `index.html` for client-side routes. The Docker
build compiles the frontend in a Node stage and copies it into
`backend/web/dist` before compiling Go.

## Consequences

- One image, one process, one port (7660), same-origin cookies, no CORS.
- Hashed assets under `/assets/` are cached for a year; `index.html` is served
  with `no-cache`.
- A plain `go build` without the frontend produces a binary that serves only
  the API and a notice page. The Makefile and Dockerfile handle the full
  build.
- During development, the Vite dev server proxies `/api` to the Go server.
