# queries — Notes for AI Agents

Custom MongoDB queries that don't fit the [service](../service/AGENTS.md) layer's standard CRUD — see [README.md](README.md). [upgrades](upgrades/) holds the per-version data migrations, [sync](sync/) holds index definitions plus the reconcile passes that make them buildable. Repo-wide rules, including the upgrade-slot rule this package enforces, are in [../AGENTS.md](../AGENTS.md).

## Every change stream runs inside `changeWatcher`, and only cancellation stops it

A MongoDB change stream ends with `Next() == false` and `Err() == nil` when the server closes it (an `invalidate`, from a dropped or renamed collection), and the driver never reopens it. A bare `for cs.Next(ctx)` loop therefore dies without a word, and that server misses every later change until it restarts. [watch.go](watch.go) reopens with backoff, resumes with `SetStartAfter` (the only resume option that survives an `invalidate`), drops a token the server refuses, and runs `onOpen` on every open to catch up on missed events. Every new watcher must use it.

Because the loop never ends on its own, whoever owns the context owns its lifetime. Domain factories start their watchers on the refresh context, and a dropped factory must be closed with `service.Factory.Close`, which cancels it (`server.removeDomain` does this), or its watchers run for the life of the process against a domain nobody serves.

## The realtime watchers see inserts and replacements, never `$set`

`WatchStreams`, `WatchUsers`, and `WatchImports` open without `UpdateLookup`, so an update written with `$set` carries no document and sends no browser nudge; only whole-document saves do. `WatchDomain` does use `UpdateLookup`, because the upgrade runner writes `databaseVersion` with `$set` and never touches the cached record, so the watcher is the only thing that brings the cache up to date. A stale cached version is harmless on its own, because every whole-document save starts from a `Domain.Load` of the stored record rather than from the cache (see [../service/AGENTS.md](../service/AGENTS.md)), but `UpgradeMongoDB` reads its starting version from the cache at boot, and a version that never arrived would re-run migrations after the next restart (see [BOOT-MIGRATIONS](../../emissary-specs/projects/BOOT-MIGRATIONS.md) §2).

## The Domain record's `_id` is the zero ObjectID

The Domain is a singleton stored with `_id: ObjectId("000000000000000000000000")`: nothing assigns `DomainID`, and `UpgradeMongoDB` filters on `primitive.NilObjectID`. The "skip zero IDs" guard in the realtime watchers must never be copied into `WatchDomain`, which would then publish nothing at all.

## Upgrade slots are append-only, and every new migration must be idempotent

[../AGENTS.md](../AGENTS.md) states the never-reuse-a-slot rule; the local mechanics live in [upgrade.go](upgrade.go), where the slice index IS the stored `databaseVersion` (slot 0 is nil). Retired migrations become no-op stubs in place (v001–v019), never deleted or renumbered. Because slot numbers were reused on dev branches in the past, version tracking is an optimization, not the safety mechanism: write every migration idempotent — plan-then-write, delete only true duplicates, backfill only nulls — so a later slot can safely re-run it on a database whose version number lies. `reconcileRules` (written for v027, re-run by v028) is the pattern.

## A new unique index must ship with the reconcile that clears its violations

`indexer.Sync` builds indexes at boot; a unique index over legacy data E11000s on every start and never builds, with no path to recovery except code. Either clean the data in a new upgrade slot before the index first ships (v027/v028), or dedupe inside the sync function itself immediately before `indexer.Sync` (`deduplicateResponses`, `deduplicateFollowing` in [sync](sync/)) with the dedupe failure reported, not fatal, so the query indexes still build.

## `parentId` never means "owner", and on Follower it is shared across types

Follower's `parentId` holds a User, Stream, OR Search ID depending on `type`, so a filter on `parentId` alone is correct only by the accident that ObjectIDs don't collide — always AND a `type` clause ([followers.go](followers.go)). CollectionItem's `parentId` holds the parent object's ID (the Stream for Likes/Replies), not the owning User's: a count or delete filter fed a UserID matches zero rows, and the failure is silent — counters refresh to 0, deletes orphan rows. Before writing any hand-built bson/exp filter that touches `userId` or `parentId`, check which identity the field actually stores on that model.

## Everything here bypasses the service layer — keep business writes out

This package exists for Mongo-only features (aggregation pipelines, raw multi-document updates, index sync, migrations) that the `data` package cannot express. Its writes skip model schema validation, service side effects, and postcommit task publication, so it is for denormalized counters, reconciliation, and migrations — not CRUD, which belongs in service. `mongoCollection` ([utils.go](utils.go)) unwraps only data-mongo collections and returns nil for anything else; per [queries.go](queries.go), this package is the seam to rewrite if the database ever changes.

## v027 keys ACTOR rules by the RAW trigger — both MatchKey shapes live on disk

Service `Rule.Save` keys ACTOR rules by the resolved canonical actor URL, but [upgrades/v027.go](upgrades/v027.go) computes `model.RuleMatchKey(record.Type, record.Trigger)` from the raw stored trigger, because a migration cannot resolve handles over the network per-row. Both shapes persist, and point lookups probe both (`loadActorRule` in [../service/rule_blocks.go](../service/rule_blocks.go)). Do not write a migration that "fixes" raw-keyed rules by resolving them — that re-introduces network calls into a migration — and see [../model/AGENTS.md](../model/AGENTS.md) for the matching-engine side of this contract.

## Nothing purges SearchResults by age

There is no global time-based purge. The daily tasks are PurgeActivityStreamCache, PurgeErrors, PurgeDomeLog, Shuffle, RecycleDomain, PurgeImports, and PurgeNotifications; hourly is PollFollowing-Index; startup queues only the Stripe Connect repair ([BUG-179](../../emissary-specs/bugs/BUG-179-Stripe-Connect-Webhook-Secret-Never-Stored.md)), which purges nothing. `queries.Recycle` only touches rows with `deleteDate > 0` older than 30 days, and SearchResults are **hard** deleted, so they never carry a deleteDate — and there is no TTL index on the collection. A SearchResult therefore lives forever unless something deletes it explicitly. Do not assume a retention window exists when reasoning about growth or about stale rows.

## An aggregation-pipeline `$set` BROADCASTS across an existing array

An update pipeline whose new value references the field it is replacing (`{$set: {loc: {type: "Point", coordinates: "$loc"}}}`) does **not** replace an array-valued field. It produces an array of N identical documents, one per original element. Only a real MongoDB shows this — reasoning about it does not. Use a plain, non-pipeline `$set` with a literal document computed in Go.

Related and still open: `idx_SearchResult_Notified` indexes `notifiedDate`, which nothing writes, so the compound index is unusable for the lock query (its leading field is always missing). It looks copied from the SearchQuery index set where the field is real. By contrast `lockId` and `timeoutDate` ARE real despite being absent from `model.SearchResult` — `queries.LockSearchResults` writes them, bypassing the model on purpose. All three are recorded in the allow-list in `sync/searchResult_test.go`.
