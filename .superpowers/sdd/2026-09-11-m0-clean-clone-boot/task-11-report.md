# Task 11: Frontend contract alignment — report

## Status
DONE

## Commit
(filled in after commit — see status line)

## What changed
- `clients/src/interfaces/user.ts`: rewritten to match `models/user.go`, `models/role.go`,
  `models/permission.go` exactly. Verified against the Go structs (not just the brief's say-so):
  `Role.Permissions []Permission` with `Permission{Id, Name}`, `User.RoleId uint`. Added
  `Permission`, widened `Role.permissions` from `string` to `Permission[]`, widened `roleId`
  from the literal `1` to `number`, and added `ApiError { code, message }`.
- `clients/src/api/client.ts` (new): single `api<T>(path, init)` wrapper. Builds the URL from
  `REACT_APP_API_URL` (default `http://localhost:8080`) + `/api/v1`, always sends
  `credentials: 'include'`, throws a typed `ApiRequestError(status, code, message)` on non-2xx
  (falling back to a synthetic body if the error response isn't JSON), and returns `undefined`
  without parsing a body on 204.
- `clients/src/pages/Login.tsx`: replaced the `axios` call and its `//@ts-ignore` with `api()`.
  Failure now sets a visible `error` state rendered as `.alert.alert-danger[role=alert]`, instead
  of throwing an unhandled rejection that silently skipped `setRedirect(true)`. Submit button is
  disabled while `submitting`. Both inputs are now controlled (`value=`) and each has a real
  (visually-hidden) `<label>` instead of relying on `placeholder` as the only label. Removed the
  `console.log`. The ADR-0001 "accounts are created by an administrator" copy was already present
  in this file pre-task and is retained as-is.
- `clients/src/Components/Layout.tsx`: the `/user` probe now goes through `api<UserInfo>('/user')`;
  `//@ts-ignore` and `axios` removed.
- `clients/src/Components/Nav.tsx`: both the `/user` fetch and the `/logout` call go through
  `api()`; both `//@ts-ignore`s removed. The `/user` fetch is now wrapped in try/catch (previously
  it had none, so every signed-out render threw an unhandled rejection). Markup fixes: each `Link`
  is now wrapped in an `<li>` (was a direct, non-`<li>` child of `<ul>` — invalid list markup);
  since that `<ul>` carried no Bootstrap list-reset class, added `navbar-nav flex-row` alongside
  the existing classes so the two items stay inline instead of becoming a bulleted stacked list —
  a consequence of fixing the markup, not a restyle. The brand `<a href="#">` became `<Link to="/">`.
- `clients/src/App.tsx`: not modified. Checked first — it already has no `Register` import or
  route (that removal predates this task), so there was nothing to do here; the brief's step 4
  turned out to be a no-op in the current tree.

## `//@ts-ignore` count
The brief says "one above every axios call" and "five total." The actual count in the pre-task
tree was **four** (`Login.tsx` ×1, `Layout.tsx` ×1, `Nav.tsx` ×2 — one per axios call, and there
were four axios calls, not five). All four are removed; `grep -rn "ts-ignore" clients/src` now
returns nothing.

## axios vs fetch
Chose `fetch` (per the brief's reference implementation) and removed all `axios` usage from
`clients/src`. Did **not** remove the `axios` dependency from `package.json`/`package-lock.json`:
running `npm uninstall axios` rewrote `package-lock.json` by ~14k lines (an npm-version lockfile
migration unrelated to this change, not just the one-package diff), which is a disproportionate
and risky diff for a frontend-contract task. Reverted that and left `axios` as an unused
`package.json` dependency instead — flagged below as a concern rather than silently left in or
silently removed.

## Verification
- `npx tsc --noEmit` — clean, no errors.
- `npm run build` (`CI=true`) — `Compiled successfully`, build output removed after checking.
- `grep -rn "ts-ignore\|axios" clients/src` — no matches.
- Confirmed the `user.ts` contract against the Go source directly: `models/user.go`,
  `models/role.go`, `models/permission.go` — `json` tags match the new interfaces field-for-field.

## Concerns
- `axios` remains in `clients/package.json` as an unused dependency. Removing it cleanly requires
  a lockfile regeneration this task's scope didn't warrant touching; left for a dedicated
  dependency-cleanup pass.
- `clients/src/App.tsx` was not touched — the registration route was already absent, so brief
  step 4 was a no-op against the current tree, not something this task did.
- No visual verification (no dev server run against a live backend) — checked `tsc`/`build`
  only, per the task's own verification bar. The Nav `navbar-nav flex-row` addition is inferred
  from Bootstrap's own class semantics to preserve the pre-existing inline layout, not visually
  confirmed in a browser.
