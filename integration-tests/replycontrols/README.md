# Reply controls, end to end

[run.sh](run.sh) starts a private Emissary server built from this checkout, federates a reply between two local domains, and checks the reply controls described in emissary-specs `projects/REPLY-CONTROLS.md` (§1 and §7) and the "Who may reply" paragraph of [service/AGENTS.md](../../service/AGENTS.md). It prints one `PASS` or `FAIL` line per check and exits 1 if any check fails, or 2 if the harness itself cannot run.

## What it tests

Domain A is `localhost` (owner `alice`), and domain B is `127.0.0.1` (owner `bob`). One server process serves both on a free port. Both names pass `uri.ValidateHostname` without editing `/etc/hosts`: `localhost` is a loopback name and `127.0.0.1` is an IP address. Because both are local, the server puts the port in every URL it builds, so federation never reaches another server on port 80.

- **1 Publish:** an open note's JSON-LD has `interactionPolicy.canReply.automaticApproval` equal to exactly the full Public IRI, declares the `gts:` terms in `@context`, and has no `commentsEnabled`. Closed, the list is exactly Alice's actor ID (1a, 1b, 1c).
- **2 Refuse at the inbox:** Bob replies to a note that is closed from the start. The reply reaches Alice's `Inbox` collection, which proves delivery, yet no REPLY notification is created, the stored `replyCount` stays 0, and `/pub/replies` is still empty after the note is reopened (2a to 2c).
- **3 Hidden, not deleted:** a reply received while open is listed in `/pub/replies` and drawn as "1 Reply" by `responses-replies`. Closed, the collection is empty and the count reads "0 Replies", while the stored `replyCount` stays 1. Reopened, both come back (3a to 3h).
- **4 Honor the remote policy:** Bob's inbox thread view (`/@me/newsfeed/browse-tree`) draws the Reply button only when the note accepts him, and the composer (`/@me/newsfeed/browse-reply`) shows either the reply form or "This post does not accept replies from you." (4a to 4f).

## How it seeds

- **Users:** each domain is configured with `createOwner: true`, so the server creates `alice` and `bob` with the localhost password `demo`. The script signs in over HTTP.
- **Notes:** Alice posts through `POST /@me/outbox-add?templateId=outbox-message`, and the script reads the new Stream's ID from the database.
- **Replies:** Bob replies through his inbox composer, `POST /@me/newsfeed/reply`, which publishes from his outbox and delivers a signed `Create` to Alice's inbox. No signature is forged by the harness.
- **Reply policy:** no embedded template offers the `allowReplies` toggle (only qwertylicious does), so the script sets `Stream.replyPolicy` in the database and then saves the note through its own `edit` form. That republish refreshes the shared ActivityStream cache, which the inbox views read.

## Prerequisites

- **MongoDB** on `mongodb://127.0.0.1:27017` as a replica set, such as the Docker container from [docker-compose-mongo.yml](../../docker-compose-mongo.yml). Set `RC_MONGO_URI` to use another; keep `?directConnection=true`.
- **Go and curl.** The script builds the server and its helper itself.

## How to run

```sh
./integration-tests/replycontrols/run.sh
```

It takes about 8 seconds with a warm build cache, plus however long `go build` needs. `RC_KEEP=1` keeps the temporary folder, with the server log and config, for inspection.

## What it touches

- **Processes:** one server on a free port, stopped on exit. It never contacts any other server.
- **Databases:** `rctest_<pid>_a`, `rctest_<pid>_b`, and `rctest_<pid>_common`, dropped on exit. A run also drops any `rctest_<pid>_*` database whose process no longer exists, left by a run that was killed. The helper refuses any database name without the `rctest_` prefix.
- **Files:** one `mktemp` folder holding the binaries, config, uploads, and log, removed on exit. Nothing in the checkout changes.

## The helper

[rchelper](rchelper/main.go) does the database work, the JSON checks, and the polling. Its `replycontrols` build tag keeps it out of `go build ./...`, `go vet ./...`, and `go test ./...`.

## Limits

- **One shared cache:** both domains share one common database, so Bob reads Alice's republished note immediately. Two real servers share nothing, and Bob's server would keep its cached copy until an `Update` reached it or the copy expired.
- **No real GoToSocial:** nothing here checks that another implementation reads the published policy.
