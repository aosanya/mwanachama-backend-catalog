# Hub copy diff

`mwanachama-backend-catalog` is the custodian of the catalog module. The hub (`projects/hub-steward/code/wakala-api/modules/catalog`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue CAT134, one of the ISS-105 moves).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-catalog]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/catalog`, relative to this repository's parent folder. The exit code is 1 when the output has any line starting with `diff `, `Binary files ` or `Only in hub`, and 0 otherwise, so `Only in backend` lines alone (this repository's own tests, `spec` and `routes/testdata`) leave it at 0. The script and `scripts/hubdiff` come from `mwanachama-backend-git` (G78) with the module path changed; the script here carries the fix of hub issue S70 (the exit rule of I13 and M6).

## What it does

1. Copies the `*.go` and `*.json` files of this repository and of the hub folder into two temporary folders, leaving out `cmd/`, `documentation/`, `scripts/` and `.git/`. The hub never had `cmd/`.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/`, and `wakalaserver/modules/catalog` becomes `github.com/aosanya/mwanachama-backend-catalog`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub` and prints its output.
5. Exits 1 if that output has a `diff ` line (a file differs, JSON files included), a `Binary files ` line or an `Only in hub` line (the hub has something this repository lacks); otherwise exits 0.

## Result of 2026-11-16

On the live hub folder the script exits 1 today, and that is correct: it prints one `diff routes/routes.go hub/routes/routes.go` (the hub copy lacks the `guardQueries` call of CAT135) and an `Only in backend: query_guard.go` line for `routes/query_guard.go`. The exit stays 1 until the hub copy gets those two changes.

On a scratch copy of the hub folder carrying both CAT135 changes there is no `diff` line and no `Only in hub` line, and the script exits 0. In that scratch copy a changed `.go` file, a changed `.json` file (a renamed key in `catalog.operations.json`) and a file only the hub has each exit 1. What remains is `Only in backend`, all of it test files, test data or the example specs the hub copy never carried:

| Difference | Reason |
|---|---|
| Only here: about 130 `*_test.go` files in the root package, `models` and `routes`, `routes/testdata` | The hub copy of 2026-11-04 was made without most test files. This repository kept them. They test code that is identical in both. |
| Only here: `spec/` (the example domain specs) | The hub builds its spec from `SpecFor`; the examples are this repository's fixtures. |
| Only here: `control_bytes_test.go` | The red test written for hub 6ea02a12 (a control byte in a staff note answers the matching invalid error); the hub commit had none. |
| Only here: `raised_item_follow_view_test.go` | The red test written for hub 0cbde117 (the follower view carries status and reply text and no staff ids); the hub commit had none. |

Backend-only tests that the hub changes made stale were updated in the commit that needed it: `raised_item_place_test.go` (a typed area is kept over a pin and an address), `routes/tools_test.go` (64 tools), and the `DeleteEntry` callers (a third argument, the deleter).

`domain_agnostic_test.go` allows three identifiers the hub named with the word agency (`agencyOwnerKey`, `AsAgencyOwner`, `isAgencyOwner`) so that the sources stay identical bar the import path. Renaming them is open as an owner question.

## The hub commits carried over

One commit here per hub commit or group, each naming the hub hash: e5220a5b, e6728d3e, 99a81288 with 2d639317, 32afdf83, f1f3ec3f, 0a259baa and 26989178, 27df7d13, f2f44c31, 6ea02a12, 0cbde117, 5c37d585, c3a8a4f8, d8836484, 1f1fcb50.

Files a commit touched lost their comments (the repository rule is no comments in code).

`record_postgres_race_test.go` (hub d8836484) runs only when `DATABASE_URL` points at localhost, and has not been run against a Postgres yet.

`TestAFloodOfRepeatedStatusesThatBreaksTheStoreNamesNothingAboutIt` failed before this work (GET /suggestions with 40000 statuses answered 200) because the Go toolchain's 10,000-parameter query cap made the dispatcher read the request as unfiltered (hub issue CAT135). The fix is `routes/query_guard.go` and one call in `routes/routes.go`, with `oversized_query_refused_test.go`. The hub copy has the same behaviour and needs the same two changes to stay identical bar the import path; until then `scripts/hub-copy-diff.sh` shows `Only in backend` for `routes/query_guard.go` and a `diff` for `routes/routes.go`, and exits 1, which is the correct answer for a real difference. The guard refuses only the cap error since hub issue CAT136, so the hub copy's `routes/query_guard.go` needs that version too until hub issue ISS-105 deletes the copy.
