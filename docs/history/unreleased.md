> [!IMPORTANT]
> **Upgrade notes.** `docker-compose.yml` runs the published image instead of
> building the checkout. Data in `./data` is unchanged.
>
> - If `.env` sets `PIKAPU_IMAGE=pikapu:latest`, the earlier example value,
>   remove the line or set a published tag such as `yexca/pikapu:latest`.
>   Then run `docker compose pull` and `docker compose up -d`.
> - The container no longer has the fixed name `pikapu`. Replace
>   `docker exec pikapu …` with `docker compose exec pikapu …`, run in the
>   directory that holds `docker-compose.yml`.
> - To keep running a local build of a checkout, use `make docker-up`.

## Operations

- `docker-compose.yml` runs `yexca/pikapu:latest` and works from an empty
  directory: no checkout, build, or `.env` file is needed. Set `PIKAPU_IMAGE`
  to pin a version or digest.
- The stack runs the container with a read-only root filesystem, a bounded
  `/tmp`, `no-new-privileges`, and no Linux capabilities except
  `DAC_OVERRIDE`, and gives it 20 seconds to shut down.
- Upgrading is `docker compose pull` followed by `docker compose up -d`.

## Development

- `deploy/compose/dev.yml` builds the checkout as `pikapu:dev` with the same
  settings as the production stack. `make docker-up`, `docker-down`,
  `docker-status`, and `docker-logs` use it.
- `make docker-config` validates both Compose files, and `make smoke` runs it
  and then tests the image under the production stack's isolation.
