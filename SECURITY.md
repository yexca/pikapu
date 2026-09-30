# Security Policy

Pikapu is a self-hosted, single-user RSS reader. This page describes its
security model and how to report a vulnerability. Deployment hardening is
covered in [docs/operations/security.md](docs/operations/security.md).

## Supported Versions

Security fixes target the latest release matching [VERSION](VERSION) and the
current main branch. Older builds are not supported.

## Security Model

- **Access control.** Without `PIKAPU_PASSWORD`, anyone who can reach the port
  can read and change everything. With it, every API route except the health
  check and sign-in requires a session cookie signed with a per-installation
  secret. Changing the password invalidates existing sessions.
- **Untrusted feeds.** Feed documents, article HTML, images, favicons, and
  redirects come from third parties. Article HTML is sanitized on the server
  before the browser renders it; only allowlisted video embeds are kept.
- **Outbound requests.** Pikapu fetches whatever feed URLs the operator
  subscribes to, including private network addresses. Treat the ability to add
  feeds as trusted, operator-level access.
- **Data.** Subscriptions, articles, reading state, and the session secret are
  stored in `data/pikapu.db`. Protect and back up that directory.

In scope: authentication bypass, script execution from feed content (XSS),
session forgery, leaking data to unauthenticated clients, or crashes and
resource exhaustion triggered by feed content.

Out of scope: attacks that require the password, instances deployed without a
password on an untrusted network, and requests to addresses the operator
subscribed to deliberately.

## Reporting a Vulnerability

Report privately to the project maintainer, for example through the hosting
platform's private vulnerability reporting, rather than a public issue.
Include the affected version (`/api/healthz` reports it), reproduction steps,
and the impact you observed. Use synthetic data; do not send real databases
or feed lists.
