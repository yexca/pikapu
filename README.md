<p align="center">
  <img src="docs/assets/pikapu-icon.png" width="112" height="112" alt="Pikapu logo">
</p>

<h1 align="center">Pikapu</h1>

<p align="center">
  A quiet, self-hosted RSS reader for one person.
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="docs/readme/README.zh-Hans.md">简体中文</a>
</p>

Pikapu collects the RSS, Atom, and JSON feeds you care about and presents them
in a clean three-pane reader. It runs as a single Docker container: a Go
backend with an embedded React frontend and a SQLite database, served on one
port.

> [!IMPORTANT]
> Pikapu is early software. Back up `data/` before upgrading. Put it behind a
> reverse proxy with TLS before exposing it to the internet.

## Key Features

- **Subscribe by URL or homepage.** Paste a feed address or just a website;
  Pikapu discovers the feed from `<link rel="alternate">` or common paths.
- **Organized reading.** Categories, unread counts, starring, search, and an
  unread/all filter across every view.
- **Keyword filters.** Mark new articles as read or skip them by keyword, for
  every feed or just one.
- **Two layouts.** A classic three-pane reader, or a hub that puts
  recommended picks first and groups new articles by day and source, like a
  notification center. Picks learn from which feeds you read and star.
- **Comfortable reader.** Three panes on desktop, a single column on phones,
  light and dark themes, adjustable text size, and keyboard shortcuts.
  Install it to your home screen or desktop to use it like an app.
- **Safe article rendering.** Feed HTML is sanitized on the server; relative
  links and lazy-loaded images are repaired; site favicons are cached.
- **Background refresh.** Conditional requests (ETag / Last-Modified),
  exponential backoff for failing feeds, and automatic cleanup of old read
  articles. Starred articles are always kept.
- **Portable subscriptions.** OPML import and export.
- **English and Simplified Chinese.** The UI follows your browser language and
  falls back to English; you can also choose a language in Settings.
- **Admin account.** Sign-in with a username and password stored as an
  Argon2id hash, a list of signed-in devices you can sign out, and rate
  limiting of failed attempts.

## Quick Start

Requires Docker with Compose.

```sh
git clone <repository-url> pikapu
cd pikapu
docker compose up -d --build
```

On first start Pikapu prints a one-time setup token to its log:

```sh
docker compose logs pikapu | grep setup_token
```

Open <http://localhost:7660>, enter the token, and create the admin account.
Then click **+** and add your first feed.

To create the account without the token instead, copy
[`.env.example`](.env.example) to `.env`, set `PIKAPU_ADMIN_USERNAME` and
`PIKAPU_ADMIN_PASSWORD` before the first start. See
[Configuration](docs/operations/configuration.md) for every option.

The default port mapping listens on all host interfaces. Bind it to
`127.0.0.1`, put it behind a reverse proxy with TLS, or use a trusted VPN if
the instance should not be reachable from your network. See
[Security](docs/operations/security.md).

## Runtime Data

| Host path | Container path | Purpose | Back up? |
| --- | --- | --- | --- |
| `./data` | `/data` | SQLite database (`pikapu.db`, plus `-wal` / `-shm` files) | Yes |

Do not commit `data/` or `.env`; they contain your subscriptions, reading
history, account, and sessions.

## Documentation

| Goal | Start here |
| --- | --- |
| Use Pikapu | [User guide](docs/user/getting-started.md) |
| Deploy and configure | [Docker](docs/operations/docker.md) · [Configuration](docs/operations/configuration.md) |
| Understand the design | [Architecture](docs/architecture/index.md) |
| Change the code | [Local development](docs/development/local-dev.md) · [Testing](docs/development/testing.md) |
| Find every document | [Documentation index](docs/README.md) |

## Development and Contributing

The `Makefile` is the canonical entry point. Tests run in Docker by default:

```sh
make help   # list targets
make test   # backend + frontend checks in containers, then a smoke test of the image
```

- [Local development](docs/development/local-dev.md)
- [Testing](docs/development/testing.md)
- [Internationalization](docs/development/i18n.md)
- [Contributing](CONTRIBUTING.md)
- [Agent guide](AGENTS.md)

## Security

Pikapu fetches untrusted feed content and renders sanitized article HTML.
Please report suspected vulnerabilities privately as described in
[SECURITY.md](SECURITY.md).
