# Deployment Security

See [SECURITY.md](../../SECURITY.md) for the security model and reporting.

## Checklist

- **Create the admin account right after the first start**, or set
  `PIKAPU_ADMIN_PASSWORD` before it. Until the account exists, only someone
  who can read the server log can claim the instance.
- **Keep production mode.** `PIKAPU_MODE=development` turns sign-in off.
- **Use TLS** in front of Pikapu for any access over an untrusted network, and
  forward `X-Forwarded-Proto: https` so the session cookie is `Secure` and
  gets the `__Host-` prefix. Set HSTS at the proxy.
- **Tell Pikapu about your proxy** with `PIKAPU_TRUSTED_PROXIES`, so sign-in
  rate limiting sees real client addresses. Without it, every client behind
  the proxy shares one limit: still safe, but an attacker can then slow down
  your own sign-in.
- **Limit exposure** where you can: bind the port to `127.0.0.1` so only the
  proxy reaches it, or use a VPN.
- **Protect `data/`.** It holds your subscriptions, articles, account, and
  sessions.

## Admin Account

- There is exactly one account: a username and a password of 8–128
  characters. The password is stored as an Argon2id hash.
- **Creating it.** Either set `PIKAPU_ADMIN_USERNAME` and
  `PIKAPU_ADMIN_PASSWORD` before the first start, or open Pikapu and enter
  the one-time setup token from the log
  (`docker compose logs pikapu | grep setup_token`). A new token is printed on
  every start until the account exists.
- **Changing it.** **Settings → Account → Edit** changes the username or
  password and asks for the current password. A new password signs out every
  other device.
- **Forgotten password.** Run `docker exec pikapu pikapu reset-password`. It
  prints a new random password and signs out every device.

## Sessions

- Sign-in sets an `HttpOnly`, `SameSite=Lax` cookie with a random token. The
  database stores only its SHA-256 hash, so a copy of `data/` cannot be used
  to sign in.
- A session ends after 30 days without use, when you sign out, or when you
  sign it out from **Settings → Account → Devices**.
- Failed sign-ins are rate-limited per client address, with waits that grow
  to 15 minutes, and across all clients. See
  [Configuration](configuration.md#fixed-limits).
- State-changing requests that a browser marks as coming from another site
  are rejected, in addition to the `SameSite` cookie.
- The web app is served with a Content Security Policy that only runs the
  app's own scripts.

## Feed Content

- Article HTML is sanitized with a strict allowlist before storage: no
  scripts, styles, event handlers, forms, or arbitrary iframes. Video embeds
  from YouTube, Vimeo, and Bilibili players are allowed; other embeds become
  links.
- Links open in a new tab with `rel="noopener noreferrer"`, and the app sends
  no `Referer` header, so sites do not learn where readers come from.
- Favicons are served with a restrictive `Content-Security-Policy` and
  `X-Content-Type-Options: nosniff`.

## Outbound Requests

Pikapu fetches the URLs you subscribe to, plus each site's home page and
favicon. It does not block private network addresses, because a personal
reader may legitimately follow feeds on a LAN. Anyone who can add feeds can
therefore make the server request internal URLs. Keep the instance
password-protected if that matters on your network.
