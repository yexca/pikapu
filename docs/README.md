# Pikapu Documentation

Public documentation, organized by reader task: using Pikapu, running it,
understanding it, and changing it.

## Start Here

- [Overview](overview.md)
- [Getting started](user/getting-started.md)
- [Docker deployment](operations/docker.md)
- [Configuration](operations/configuration.md)
- [Troubleshooting](operations/troubleshooting.md)

## By Area

- **User guide**
  - [Getting started](user/getting-started.md)
  - [Reading](user/reading.md): views, filters, search, shortcuts
  - [Subscriptions](user/subscriptions.md): adding feeds, categories, OPML,
    update errors
  - [Filters](user/filters.md): marking as read or skipping articles by keyword
  - [Settings](user/settings.md): language, appearance, refresh, retention
- **Operations**
  - [Docker](operations/docker.md)
  - [Configuration](operations/configuration.md)
  - [Security](operations/security.md)
  - [Troubleshooting](operations/troubleshooting.md)
- **Architecture**
  - [Index](architecture/index.md)
  - [Backend](architecture/backend.md)
  - [Frontend](architecture/frontend.md)
  - [Data model](architecture/data-model.md)
  - [HTTP API](architecture/api.md)
- **Development**
  - [Local development](development/local-dev.md)
  - [Testing](development/testing.md)
  - [Internationalization](development/i18n.md)
  - [Commit and release](development/commit-and-release.md)
- [Decisions](decisions/index.md): architecture decision records
- [History](history/index.md): release notes

## Reading Paths

New users: [Overview](overview.md) → [Getting started](user/getting-started.md)
→ [Reading](user/reading.md).

Operators: [Docker](operations/docker.md) →
[Configuration](operations/configuration.md) →
[Security](operations/security.md).

Developers: [Agent guide](../AGENTS.md) → [Architecture](architecture/index.md)
→ [Testing](development/testing.md) → [Internationalization](development/i18n.md).

## Documentation Rules

- Documentation is written in English. Translated READMEs live in
  [`readme/`](readme/).
- User-visible behavior belongs in `user/`, runtime instructions in
  `operations/`, system boundaries in `architecture/`, developer workflow in
  `development/`, and durable design choices in `decisions/`.
- Record user-facing changes in [history/unreleased.md](history/unreleased.md).
