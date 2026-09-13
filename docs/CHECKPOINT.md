# Milestone checkpoint

A milestone is not done when the tests pass. It is done when a human has run it
and seen it work, and when the repository is fit to be public.

Run this before opening a PR for any milestone. Record what you actually
observed, not what you expected.

---

## 1. Boot from genuinely nothing

```bash
docker compose -f deploy/compose/docker-compose.yml down -v
make up
```

`down -v` is not optional. Postgres init scripts run **only on an empty
volume**, so a stale volume silently keeps old roles and masks mistakes in
exactly the files you are trying to verify.

Every container must report `(healthy)`, not merely `Up`:

```bash
docker compose -f deploy/compose/docker-compose.yml ps
```

A service that only ever reaches `Up` has no working health probe, which means
`depends_on` is lying to everything downstream of it.

## 2. Drive the product path by hand

Do not skip this because the integration test covers it. The test asserts what
someone already thought to assert; you are looking for what nobody did.

```bash
curl -s -H 'X-Request-Id: checkpoint' localhost:8080/_platform/identity
curl -s -H 'X-Request-Id: checkpoint' localhost:8080/_platform/catalog
curl -s -H 'X-Request-Id: checkpoint' localhost:8080/_platform/orders
```

Each must return **its own** service name, a non-zero `schemaVersion`, and echo
back the request ID you sent. A wrong service name means the gateway routed
incorrectly; a zero version means migrations did not run; a different request
ID means correlation is broken.

## 3. Break it on purpose

A system that only works is not yet understood. Confirm the failure paths:

```bash
# unknown route -> standard envelope, never a stack trace
curl -s localhost:8080/_platform/nope

# liveness must not depend on the database; readiness must
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U identity_user -d identity_db -c "UPDATE schema_migrations SET dirty = true;"

curl -s localhost:8080/_platform/identity                        # 503, schema_dirty
curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/readyz   # 503
curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/healthz  # 200 — still alive

docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U identity_user -d identity_db -c "UPDATE schema_migrations SET dirty = false;"

# database isolation is real, not aspirational
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql "postgres://identity_user:dev_only_identity@localhost/catalog_db" -c 'SELECT 1'
# -> FATAL: permission denied for database "catalog_db"
```

If `/healthz` fails alongside `/readyz`, liveness is wrongly touching the
database — a defect to report, not to work around by relaxing the check.

## 4. Full suite

```bash
make lint
make test
```

## 5. Public-repository audit

This repository is public. Run this before every PR.

```bash
# nothing from the working scratch directories may be tracked
git ls-files | grep -E '^\.superpowers|^\.claude' && echo "LEAK" || echo "clean"

# No internal hostnames or package mirrors. Two checks, deliberately:
#
#   (a) an explicit denylist, case-insensitive. A generic hostname regex was
#       tried first and silently failed to match gitlab.<employer>.com — the
#       one string this check exists for. Add terms as they come up.
#   (b) a hostname *shape*, case-SENSITIVE: matching case-insensitively turns
#       Go identifiers like errs.Internal.Wrap into false positives.
#
# The filter drops the public branch name fix/remove-internal-registry-paths.

# This file is excluded: it contains the denylist terms itself, and an audit
# that always fires is one people learn to ignore.
git grep -rin 'nordstrom\|artifactory' -- . ':!docs/CHECKPOINT.md' \
  | grep -v 'fix/remove-internal-registry-paths' && echo "LEAK (a)" || echo "clean (a)"

git grep -rn '[a-z0-9-]\+\.\(corp\|internal\|intra\)\.[a-z]\{2,\}' -- . \
  && echo "LEAK (b)" || echo "clean (b)"

# no credentials beyond the deliberate dev_only_* local ones
git grep -rinE '(password|secret|token|api[_-]?key)\s*[:=]' -- . \
  | grep -v dev_only_ | grep -v _test.go
```

Committed `dev_only_*` passwords are deliberate: the stack must run from a
clean clone, and Postgres is bound to `127.0.0.1` so they are not reachable off
the machine. A deployed configuration must not inherit that pattern — see the
README.

Note that a file removed from the tree still lives in history. Scrubbing the
current tree reduces discoverability; it does not remove anything already
pushed.

## 6. Record what you found

Anything that failed, surprised you, or only worked on the second try belongs
in the milestone's notes — especially if you then fixed it. A checkpoint that
only ever records success is not being run honestly.
