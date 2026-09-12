# Task 6 report: Replicate the template to Catalog and Orders

**Status:** Complete. Mechanical copy of `services/identity/` to `services/catalog/` and `services/orders/`, with the corrections from the parent's brief-corrections applied (case-insensitive rename, asserted port substitutions, `router.go` copied verbatim rather than re-derived from the brief's incorrect middleware prose).

**Commit:** see below (recorded after commit).

**Test summary:** `go test ./services/... ./platform/... -count=1` — all 16 packages pass, 0 failures. Each of the three services (identity, catalog, orders) has the identical per-package RUN count: `cmd/<svc>` 2, `internal/config` 2, `internal/httperr` 3, `internal/platformcheck` 6 → 13 tests per service, 39 total across the three, matching the "same test count as Identity" requirement (the brief's "twelve tests / four per service" is stale from before Task 5's review additions).

## What was done

1. Copied `services/identity/.` to `services/catalog/` and `services/orders/`.
2. Ran `sed 's/identity/$svc/g'` (lowercase) then `sed 's/Identity/$Svc/g'` (capitalised) over every file in each copy — the brief's original `sed` only handled lowercase and would have left `services/*/openapi/*.yaml:3` reading `title: Identity Service`.
3. Renamed `cmd/identity` → `cmd/catalog` / `cmd/orders`, and `openapi/identity.yaml` → `openapi/catalog.yaml` / `openapi/orders.yaml` (the content substitution does not rename files/directories).
4. Applied port substitutions and asserted each one rather than trusting silent `sed`:
   - `grep -q 'DefaultPort = "8082"' services/catalog/internal/config/config.go` → OK
   - `grep -q 'EXPOSE 8082' services/catalog/Dockerfile` → OK
   - `grep -q 'DefaultPort = "8083"' services/orders/internal/config/config.go` → OK
   - `grep -q 'EXPOSE 8083' services/orders/Dockerfile` → OK
5. `gofmt -w services` (idempotent — `services/identity` untouched by the run, confirmed via `git status services/identity` showing nothing).

## Verification commands and output

```
$ gofmt -l services platform
(empty — clean)

$ go vet ./services/... ./platform/...
(empty — clean)

$ go build ./...
(empty — clean, whole repo including legacy root)

$ grep -rni identity services/catalog services/orders
(no output — no case-insensitive "identity" residue in either copy)

$ go test ./services/... ./platform/... -count=1 -v
... 16 "ok" lines, no FAIL
```

Per-package test counts extracted from the `-v` run (`=== RUN` occurrences per package):

```
services/catalog/cmd/catalog          2
services/catalog/internal/config      2
services/catalog/internal/httperr     3
services/catalog/internal/platformcheck 6
services/identity/cmd/identity        2
services/identity/internal/config     2
services/identity/internal/httperr    3
services/identity/internal/platformcheck 6
services/orders/cmd/orders            2
services/orders/internal/config       2
services/orders/internal/httperr      3
services/orders/internal/platformcheck 6
```
13 per service, identical across all three.

### Import-path check (binding constraint 1)

```
$ grep -rn "go-admin/services/identity" services/catalog services/orders
(no output)
```
All internal imports in `services/catalog` and `services/orders` resolve to their own service path (`go-admin/services/catalog/...` / `go-admin/services/orders/...`); no cross-service or `services/identity` imports survived the rename.

```
$ grep -rln "go-admin/controllers\|go-admin/models\|go-admin/routes\|go-admin/middlewares\|go-admin/database\|go-admin/utils\b" services/catalog services/orders services/identity
(no output)
```
No imports of the legacy root application.

### Byte-identical-apart-from-substitutions check (normalized diff)

For each of catalog/orders, every file in `services/identity` was mapped to its renamed counterpart (`cmd/identity`→`cmd/<svc>`, `identity.yaml`→`<svc>.yaml`), both sides had the substituted tokens (`identity`/`Identity` on the template side, `<svc>`/`<Svc>` on the copy side, and `8081` vs `8082`/`8083`) masked to a common placeholder, then diffed:

```
for svc in catalog orders; do
  Svc="$(tr '[:lower:]' '[:upper:]' <<< ${svc:0:1})${svc:1}"
  find services/identity -type f | sort | while read f; do
    g=$(echo "$f" | sed "s#services/identity#services/$svc#; s#/cmd/identity#/cmd/$svc#; s#identity\.yaml#$svc.yaml#")
    [ -f "$g" ] || { echo "MISSING: $g"; continue; }
    diff <(sed "s/identity/SVC/g; s/Identity/SVC/g; s/8081/PORT/g" "$f") \
         <(sed "s/$svc/SVC/g; s/$Svc/SVC/g; s/808[23]/PORT/g" "$g") >/dev/null || echo "DIFF: $f vs $g"
  done
done
```
Result: no `MISSING` or `DIFF` lines for either service. A plain `diff -rq services/identity services/catalog|orders` (unnormalized) additionally confirms that `handler.go`, `platformcheck.go`, `postgres.go`, `postgres_test.go`, `repository.go`, `router.go`, `router_test.go`, `main.go`, and `000001_init.down.sql` are byte-for-byte identical across all three services — no drift outside the documented substitution points.

### Comment/prose accuracy check (binding constraint 2)

Read `openapi/*.yaml` description and `migrations/000001_init.up.sql` comment for both new services: both are generic walking-skeleton prose ("M1 exposes only infrastructure endpoints. Domain endpoints arrive in M2.", `COMMENT ON SCHEMA public IS '<svc> service'`) with no identity-specific claims (e.g. no mention of "users," "sessions," or anything identity-domain-specific), so nothing was left false by the rename.

### git status

```
$ git status
Untracked files:
	services/catalog/
	services/orders/
```
36 new files total (18 per service), `services/identity/` shows no modifications.

## Concerns

1. The brief's Step 3 says "twelve tests (four per service)" — stale; actual is 13 per service (39 total) because of Task 5's review additions (`config_test.go`'s `DefaultPort` assertion, `postgres_test.go`, the `router_test.go` startup-contract test, `httperr_test.go`). Verified all three services carry the same count, which is what actually matters.
2. Did not check whether any Makefile/CI workflow/deploy manifest at the repo root currently hardcodes `services/identity` only (e.g. build/test targets enumerating just Identity) — out of scope for this task's file set, but worth a follow-up grep (`grep -rln "services/identity\|cmd/identity" --exclude-dir=services --exclude-dir=.git .`) before Catalog/Orders are expected to build/deploy in CI.
3. No Docker, no `test/integration/`, no migration execution was run, per the task's constraints — `migrations/` and `Dockerfile` in both new services are unexecuted files only.
