# go-admin

Staff back-office for a small e-commerce shop: a gateway in front of three Go
services — Identity, Catalog, Orders — each owning its own PostgreSQL database
and its own restricted role.

## Run

Go 1.27 and Docker. No `.env` step — local config lives in
`deploy/compose/docker-compose.yml`.

```bash
make up
make seed
curl -s -c /tmp/j -X POST localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"owner@example.com","password":"dev_only_owner_password"}'
curl -s -b /tmp/j localhost:8080/api/v1/me
curl -s -b /tmp/j -X POST localhost:8080/api/v1/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"desk-1","title":"Walnut desk","priceMinor":14999}'
curl -s -b /tmp/j localhost:8080/api/v1/products
curl -s -b /tmp/j -X POST localhost:8080/api/v1/products/1/activate
curl -s -b /tmp/j -X POST localhost:8080/api/v1/customers \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","name":"Jane Doe"}'
curl -s -b /tmp/j -X POST localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' \
  -d '{"customerId":"1","items":[{"productId":"1","quantity":1}]}'
curl -s -b /tmp/j localhost:8080/api/v1/orders
```

`make up` boots postgres, the migrate jobs, the three services and the gateway,
and waits for all of them to report healthy. `make seed` creates the first owner
account and leaves an existing one alone. `make help` lists the rest.

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

Identity is complete: login and sessions, staff, roles and permissions, all
behind route-level authorization. See `services/identity/openapi/identity.yaml`
for the contract.

Catalog is complete: products, search, images and an internal resolve endpoint
for Orders, all behind route-level authorization. See
`services/catalog/openapi/catalog.yaml` for the contract.

Orders is complete: customers, orders, and their lifecycle transitions
(cancel, refund, and the generic status change), all behind route-level
authorization. Each order line is resolved against Catalog and snapshotted at
creation time, so a later change to a product never changes what was bought.
See `services/orders/openapi/orders.yaml` for the contract.

## Dev credentials

The passwords in the compose file (`dev_only_*`) are committed so the stack runs
from a clean clone. Each role is `NOSUPERUSER NOCREATEDB NOCREATEROLE` and can
reach only its own database — `identity_user` cannot connect to `catalog_db`
(`deploy/compose/postgres/init/01-roles-and-databases.sql`) — and postgres is
bound to `127.0.0.1`.

A deployed configuration must not inherit this. DSNs sit in compose
`environment:` blocks, readable via `docker inspect`; deployed secrets need a
real secret store and different passwords.
