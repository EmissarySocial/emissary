# Reply controls, end to end

[run.sh](run.sh) starts a private Emissary server built from this checkout, federates replies and quote requests between two local domains, and checks the reply controls described in emissary-specs `projects/REPLY-CONTROLS.md` (§1 and §7), the FEP-044f quote approvals of BUG-286, and the "Who may reply" and "Quotes are approved at the inbox" paragraphs of [service/AGENTS.md](../../service/AGENTS.md). It prints one `PASS` or `FAIL` line per check and exits 1 if any check fails, or 2 if the harness itself cannot run.

## What it tests

Domain A is `localhost` (owner `alice`), and domain B is `127.0.0.1` (owner `bob`). One server process serves both on a free port. Both names pass `uri.ValidateHostname` without editing `/etc/hosts`: `localhost` is a loopback name and `127.0.0.1` is an IP address. Because both are local, the server puts the port in every URL it builds, so federation never reaches another server on port 80.

- **1 Publish:** an open note's JSON-LD has `interactionPolicy.canReply.automaticApproval` and `interactionPolicy.canQuote.automaticApproval` each equal to exactly the full Public IRI, declares the `gts:` terms in `@context`, and has no `commentsEnabled`. Closed, both lists are exactly Alice's actor ID (1a, 1b, 1c).
- **2 Refuse at the inbox:** Bob replies to a note that is closed from the start. The reply reaches Alice's `Inbox` collection, which proves delivery, yet no REPLY notification is created, the stored `replyCount` stays 0, and `/pub/replies` is still empty after the note is reopened (2a to 2c).
- **3 Hidden, not deleted:** a reply received while open is listed in `/pub/replies` and drawn as "1 Reply" by `responses-replies`. Closed, the collection is empty and the count reads "0 Replies", while the stored `replyCount` stays 1. Reopened, both come back (3a to 3h).
- **4 Honor the remote policy:** Bob's inbox thread view (`/@me/newsfeed/browse-tree`) draws the Reply button only when the note accepts him, and the composer (`/@me/newsfeed/browse-reply`) shows either the reply form or "This post does not accept replies from you." (4a to 4f).
- **5 Quote posts:** Bob sends a `QuoteRequest` for an open note. Alice's server answers with an `Accept` whose `result` is a `QuoteAuthorization` that passes the checks in Mastodon's `verify_quote_service.rb`, notifies Alice, lists the quote on her Quotes tab, and draws a Reject button in that row's modal. A request whose quote post is attributed to someone else is refused with a 422. Closed, the authorization answers 404, and a new request gets no answer and no notification; reopened, the authorization is served again. Alice's Reject button, which names the CollectionItem that approves the quote, deletes the notification and withdraws the authorization. A queued task then loads the quote post from Bob's server to learn who quoted, and delivers a `Delete` of the authorization to Bob. The quote post is a real post of Bob's for that reason (5a to 5m).
- **6 Notification tabs:** the All list holds the quote and the reply, and Messages & Replies leaves the quote out. A GET of `read-one` marks nothing read; a POST marks that one row read and leaves the others unread. Each of the six tab pages renders (6a to 6g).

## How it seeds

- **Users:** each domain is configured with `createOwner: true`, so the server creates `alice` and `bob` with the localhost password `demo`. The script signs in over HTTP.
- **Notes:** Alice posts through `POST /@me/outbox-add?templateId=outbox-message`, and the script reads the new Stream's ID from the database.
- **Replies:** Bob replies through his inbox composer, `POST /@me/newsfeed/reply`, which publishes from his outbox and delivers a signed `Create` to Alice's inbox. No signature is forged by the harness.
- **Quote requests:** Emissary does not send quotes yet, so the helper builds each `QuoteRequest` itself and signs it with Bob's own key, read from his database. The quote post is inlined, as Mastodon does, and never exists on Bob's server; nothing on Alice's side fetches it.
- **Reply policy:** no embedded template offers the `allowReplies` toggle (only qwertylicious does), so the script sets `Stream.replyPolicy` in the database and then saves the note through its own `edit` form. That republish refreshes the shared ActivityStream cache, which the inbox views read.

## Prerequisites

- **MongoDB** on `mongodb://127.0.0.1:27017` as a replica set, such as the Docker container from [docker-compose-mongo.yml](../../docker-compose-mongo.yml). Set `RC_MONGO_URI` to use another; keep `?directConnection=true`.
- **Go and curl.** The script builds the server and its helper itself.

## How to run

```sh
./integration-tests/replycontrols/run.sh
```

A run with a warm build cache took 45 seconds on 2026-10-29, including both builds. `RC_KEEP=1` keeps the temporary folder, with the server log and config, for inspection.

## What it touches

- **Processes:** one server on a free port, stopped on exit. It never contacts any other server.
- **Databases:** `rctest_<pid>_a`, `rctest_<pid>_b`, and `rctest_<pid>_common`, dropped on exit. A run also drops any `rctest_<pid>_*` database whose process no longer exists, left by a run that was killed. The helper refuses any database name without the `rctest_` prefix.
- **Files:** one `mktemp` folder holding the binaries, config, uploads, and log, removed on exit. Nothing in the checkout changes.

## The helper

[rchelper](rchelper/main.go) does the database work, the JSON checks, and the polling. Its `replycontrols` build tag keeps it out of `go build ./...`, `go vet ./...`, and `go test ./...`.

## Limits

- **One shared cache:** both domains share one common database, so Bob reads Alice's republished note immediately. Two real servers share nothing, and Bob's server would keep its cached copy until an `Update` reached it or the copy expired.
- **No real GoToSocial or Mastodon:** nothing here checks that another implementation reads the published policy, or verifies the `QuoteAuthorization` itself. Check 5c applies Mastodon's rules from a copy in the helper.
