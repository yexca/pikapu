# ADR-0004: One Admin Account With Server-Side Sessions

## Status

Accepted. Replaces the optional `PIKAPU_PASSWORD`.

## Context

Pikapu was protected by an optional shared password from the environment.
Sessions were stateless cookies signed with HMAC over an expiry and a hash of
the password. That was enough on a home network, but not for an instance on
the internet:

- The default, no password, left the whole API open.
- A single session could not be revoked; only changing the environment
  variable and recreating the container signed anyone out.
- There was no username, no way to change the password from the app, and no
  limit on guessing beyond a fixed delay.
- Cross-site requests were stopped only by `SameSite=Lax`, which does not
  cover sibling subdomains.

Pikapu remains a single-user reader ([Overview](../overview.md#non-goals)), so
multi-user accounts are not wanted.

## Decision

- **One admin account** stored in the database: a username and an Argon2id
  password hash. It is managed in Settings; there are no other users or
  roles.
- **Sign-in is always on in production.** `PIKAPU_MODE` defaults to
  `production`. `development` turns sign-in off for local previews and is
  the default only for `make backend-run`.
- **First run** needs proof of server access: either bootstrap credentials in
  the environment (`PIKAPU_ADMIN_USERNAME`, `PIKAPU_ADMIN_PASSWORD`) or a
  random setup token printed to the server log. Without one of them, nobody
  who merely reaches the port can claim a fresh instance.
- **`PIKAPU_PASSWORD` is removed** without a migration path. Existing
  instances create their account once with the setup token or the new
  variables.
- **Server-side sessions**: random tokens in `HttpOnly` cookies, stored as
  SHA-256 hashes. They can be listed and revoked, expire after 30 days
  without use, and a password change revokes all but the current one.
- **Defenses in the app rather than only at a proxy**: per-client and global
  rate limiting of failed credential checks (in memory, as there is one
  instance), `http.CrossOriginProtection` on state-changing requests, and a
  Content Security Policy for the web app.
- **Recovery** through `pikapu reset-password` inside the container, which
  sets a random password and revokes all sessions.

## Consequences

- An instance can be put on the internet behind a TLS proxy without extra
  auth layers. Reverse proxies should be listed in `PIKAPU_TRUSTED_PROXIES`
  so rate limiting can tell clients apart.
- Every authenticated request reads the `sessions` table; activity is
  written back at most hourly to keep writes low.
- Upgrading from a password-protected instance requires creating the account
  once.
- The rate limiter resets on restart, and a flood of failures from many
  addresses can delay the owner's sign-in; both are accepted for a
  single-user deployment.
- Adding more users later would need user IDs on the data tables and a
  new ADR; nothing here prepares for that.
