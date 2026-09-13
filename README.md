# go-admin

Staff back-office for a small e-commerce shop: a gateway in front of three Go
services — Identity, Catalog, Orders — each owning its own PostgreSQL database
and its own restricted role.

It replaces a 2022 monolith, which still sits at the repo root, runs on its own
MySQL stack, is tagged `legacy-v1`, and is deleted as the services take over.
Don't confuse it with `services/`.

## Run

Go 1.27 and Docker. No `.env` step — local config lives in
`deploy/compose/docker-compose.yml`.

```bash
make up
curl -s -H 'X-Request-Id: check' localhost:8080/_platform/identity
# {"service":"identity","schemaVersion":1,"requestId":"check"}
```

`make up` boots postgres, the migrate jobs, the three services and the gateway,
and waits for all of them to report healthy. `make help` lists the rest.

| gateway | identity | catalog | orders | postgres |
|---|---|---|---|---|
| 8080 | 8081 | 8082 | 8083 | 5433 (`127.0.0.1` only) |

## Test

```bash
make test          # unit + arch + integration
make lint
```

Integration tests are behind `//go:build integration`. Run them through
`make test-integration`, not by hand: a broader path without the tag silently
runs zero of them and still reports `ok`.

## What exists

The stack boots, migrates, and routes. There is no domain functionality yet —
no login, products, orders, staff or customers.

`GET /_platform/{service}` returns `{service, schemaVersion, requestId}`. It is
scaffolding so there is something to verify end to end, not a product API, and
it is removed per service as real routes land. Don't build against it.

## Dev credentials

The passwords in the compose file (`dev_only_*`) are committed so the stack runs
from a clean clone. Each role is `NOSUPERUSER NOCREATEDB NOCREATEROLE` and can
reach only its own database — `identity_user` cannot connect to `catalog_db`
(`deploy/compose/postgres/init/01-roles-and-databases.sql`) — and postgres is
bound to `127.0.0.1`.

A deployed configuration must not inherit this. DSNs sit in compose
`environment:` blocks, readable via `docker inspect`; deployed secrets need a
real secret store and different passwords.
