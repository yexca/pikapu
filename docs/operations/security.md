# Deployment Security

See [SECURITY.md](../../SECURITY.md) for the security model and reporting.

## Checklist

- **Set a password** with `PIKAPU_PASSWORD` unless the port is reachable only
  from your own machine.
- **Limit exposure.** Bind the port to `127.0.0.1`, use a VPN, or put Pikapu
  behind a reverse proxy.
- **Use TLS** in front of Pikapu for any access over an untrusted network, and
  forward `X-Forwarded-Proto: https` so the session cookie is marked `Secure`.
- **Protect `data/`.** It holds your subscriptions, articles, and the session
  signing secret.

## Authentication

- One shared password; there are no user accounts.
- Sign-in sets an `HttpOnly`, `SameSite=Lax` cookie valid for 30 days. It is
  signed with HMAC-SHA256 using a random secret generated on first start and
  stored in the database.
- The signature also covers the password, so changing `PIKAPU_PASSWORD` and
  recreating the container signs everyone out.
- Failed sign-ins are delayed by half a second. For internet-facing instances,
  add rate limiting at the reverse proxy.

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
