# Security Policy

Pikapu is a self-hosted, single-user RSS reader. This page describes its
security model and how to report a vulnerability. Deployment hardening is
covered in [docs/operations/security.md](docs/operations/security.md).

## Supported Versions

Security fixes target the latest release matching [VERSION](VERSION) and the
current main branch. Older builds are not supported.

## Security Model

- **Access control.** One admin account protects every API route except the
  health check and the sign-in and setup endpoints. Until the account exists,
  it can only be created with a setup token printed to the server log.
  Sessions are random tokens stored hashed on the server and can be revoked;
  changing the password signs out other devices. Failed sign-ins are
  rate-limited. `PIKAPU_MODE=development` turns all of this off and is meant
  only for local development.
- **Untrusted feeds.** Feed documents, article HTML, images, favicons, and
  redirects come from third parties. Article HTML is sanitized on the server
  before the browser renders it; only allowlisted video embeds are kept.
- **Outbound requests.** Pikapu fetches whatever feed URLs the operator
  subscribes to, including private network addresses. Treat the ability to add
  feeds as trusted, operator-level access.
- **Data.** Subscriptions, articles, reading state, the account's password
  hash, and session token hashes are stored in `data/pikapu.db`. Protect and
  back up that directory.

In scope: authentication bypass, claiming an instance without the setup
token, script execution from feed content (XSS), session forgery or
fixation, cross-site request forgery, bypassing the sign-in rate limit,
leaking data to unauthenticated clients, or crashes and resource exhaustion
triggered by feed content.

Out of scope: attacks that require the password or access to the server log
or `data/`, instances running in development mode, and requests to addresses
the operator subscribed to deliberately.

## Reporting a Vulnerability

Report privately to the project maintainer, for example through the hosting
platform's private vulnerability reporting, rather than a public issue.
Include the affected version (`/api/healthz` reports it), reproduction steps,
and the impact you observed. Use synthetic data; do not send real databases
or feed lists.
