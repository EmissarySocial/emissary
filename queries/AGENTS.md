# queries — Notes for AI Agents

Custom MongoDB queries that don't fit the [service](../service/AGENTS.md) layer's standard CRUD; see [README.md](README.md). [upgrades](upgrades/) holds the per-version migrations, and [sync](sync/) the index definitions plus the reconcile passes that make them buildable. Repo-wide rules, including the upgrade-slot rule, are in [../AGENTS.md](../AGENTS.md).

## Every change stream runs inside `changeWatcher`, which reopens it until canceled

When the server closes a change stream (an `invalidate`, from a dropped or renamed collection), `Next()` returns false with no error and the driver never reopens it, so a bare `for cs.Next(ctx)` loop dies silently and that server misses every later change until it restarts. [watch.go](watch.go) reopens with backoff, resumes with `SetStartAfter` (the only resume option that survives an `invalidate`), drops a token the server refuses, and runs `onOpen` on every open to catch up on missed events. Every new watcher must use it.

The loop ends only when its context is canceled, or at once on a standalone server, which cannot open a change stream. So whoever owns the context owns the watchers: a domain factory starts them on its refresh context, and `service.Factory.Close` cancels it (`server.removeDomain` calls it). A factory dropped without `Close` keeps its watchers running against a domain nobody serves.

## The realtime watchers see inserts and replacements, never `$set`

`WatchStreams`, `WatchUsers`, and `WatchImports` open without `UpdateLookup`, so a `$set` update carries no document and sends no browser nudge; only whole-document saves do. `WatchDomain` uses `UpdateLookup`, because the upgrade runner and some slots write the Domain record with `$set` and never touch the cache, so the watcher is the only thing that brings every server's cache up to date. A stale cache cannot reset the stored version today: every whole-document save starts from a `Domain.Load` (see [../service/AGENTS.md](../service/AGENTS.md)), and `UpgradeMongoDB` reads its starting version just after `Start` loads the record. [BOOT-MIGRATIONS](../../emissary-specs/projects/BOOT-MIGRATIONS.md) §2 records the reset that happened before both held.

## The Domain record's `_id` is the zero ObjectID

The Domain is a singleton stored with `_id: ObjectId("000000000000000000000000")`: no constructor mints `DomainID`, and `UpgradeMongoDB` filters on `primitive.NilObjectID`. Never copy the realtime watchers' "skip zero IDs" guard into `WatchDomain`, or it publishes nothing.

## Upgrade slots are append-only, and every new migration must be idempotent

In [upgrade.go](upgrade.go) the slice index is the stored `databaseVersion` (slot 0 is nil); [../AGENTS.md](../AGENTS.md) has the never-reuse rule. A retired migration becomes a no-op stub in place (v001–v019), never deleted or renumbered. Slot numbers were reused on dev branches in the past, so version tracking is an optimization, not the safety mechanism: make every migration idempotent (plan then write, delete only true duplicates, backfill only nulls) so a later slot can re-run it on a database whose version lies. `reconcileRules`, written for v027 and re-run by v028, is the pattern.

## Uncommitted is not unrun

A slot runs the first time any server boots with it, including a dev server running whatever is on disk in a worktree. Slot 37 was edited in place on 2026-10-11 while uncommitted, but local dev databases had already recorded it, so a slot 38 had to repair both versions. Once a slot exists on disk, treat it as run and correct it in the next slot. Folding a correction back is possible only before release, and only by also resetting `databaseVersion` on every local database that ran the later slot, as when slot 38 was merged into 37.

## Version39 is written but not registered

[upgrades/v039.go](upgrades/v039.go) copies the legacy `Outbox` collection into `OutboxItem` (OUTBOX2-OMNIBUS §1.21), but [upgrade.go](upgrade.go) does not list it. A slot's place in the list is its version, and slot 38 is I18N's `Version38`, which is only on `dev-Eugene`. Listed straight after `Version37`, it would run as slot 38 on every server, and I18N's slot would then never run anywhere. Register it directly after `Version38` once that slot is on `dev`. Until then nothing runs it, and the outbox serves no activity published before Outbox2.

It reads three shapes of row: the current one, v0.7.0's (`parentId`, `parentType`, and `url` for the object), and rows with an empty `actorUrl` (BUG-146), whose URL it looks up from the User or Stream. BUG-304's repair will take a later slot, so this one cannot rely on it. A row with no permissions was public, as slot 19 recorded, but a null list was served to no one and stays private. The object stays a URL, as the legacy outbox served it (Ben, 2026-11-19), so Funkwhale cannot import an older song from these rows.

## A new unique index must ship with the reconcile that clears its violations

`indexer.Sync` builds indexes at boot, so a unique index over duplicate legacy data fails with E11000 on every start and never builds until code changes. Either clean the data in an upgrade slot before the index first ships (v027/v028), or dedupe in the sync function just before `indexer.Sync` (`deduplicateAnnotations`, `deduplicateFollowing`, `deduplicateResponses` in [sync](sync/)), reporting a failed dedupe rather than returning it, so the other indexes still build.

A failed build also costs the index it replaced: `indexer.Sync` drops a changed or renamed index before creating the new one, and only reports a failed create, so until the dedupe succeeds every lookup on those keys scans the collection, without an error.

`idx_OutboxItem_URL` ([sync/outboxItem.go](sync/outboxItem.go)) needs no reconcile only while every writer derives `url` from the row's own `_id` through `Locator.ActivityURL`, as `Outbox2.Save` does (`TestOutboxItemIndex_BuildsOverExistingRows`). Its partial filter (`url > ""`) leaves out the empty string that `ActivityURL` returns for an unknown actor type. `Version39` stores each legacy row's old `ActivityPubURL()` as `url`, which is the row's `activityUrl` when it has one, not a URL derived from its `_id`. So the slot keeps the oldest row for each URL itself and reports the rest, because on a database upgrading from v0.7.0 this index is built only after the slot runs (`TestVersion39_SharedURLWithoutTheIndex`).

The `OutboxItem` collection has no recycle index and is not in `factory.Collections()`, because `service.Outbox2` only hard-deletes. If anything ever soft-deletes an `OutboxItem`, add `deleteDate: 0` to the unique index's partial filter and the collection to `Collections()`.

`service.Outbox2` wrote to a collection named `Outbox2` until 2026-11-17, and `origin/main` still does. `sync.OutboxItem` moves any rows left there on every start, before it syncs the indexes. It copies each row with `$setOnInsert`, so a row already in `OutboxItem` wins, and it deletes each old copy by `_id`, never by dropping the collection, so a row that an older server writes during a rolling deploy moves on the next start. It is a sync step, not an upgrade slot, because slot 38 was already claimed twice (I18N and BUG-304) and a slot runs once, which misses those late rows. The empty `Outbox2` collection stays until OUTBOX2-OMNIBUS §1.30 drops the legacy collections.

`idx_Stream_Token` deliberately became unique with no reconcile (Ben, 2026-11-03). A Domain where two live Streams share a token, or more than one has none, has no token index until those Streams are fixed by hand.

## `parentId` never means "owner", and on Follower it is shared across types

Follower's `parentId` holds a User, Stream, or Search ID depending on `type`, so a filter on `parentId` alone is correct only because ObjectIDs don't collide; always add a `type` clause ([followers.go](followers.go)). CollectionItem's `parentId` holds the parent object (the Stream, for Likes and Replies), not the owning User, so a filter fed a UserID silently matches nothing: counters refresh to 0 and deletes orphan rows. Before hand-building a filter on `userId` or `parentId`, check which identity that model stores there.

## Everything here bypasses the service layer, so keep business writes out

This package holds what the `data` package cannot express: aggregation pipelines, raw multi-document updates, index sync, and migrations. Its writes skip schema validation, service side effects, and postcommit tasks, so use it for denormalized counters, reconciliation, and migrations, never CRUD. `mongoCollection` ([utils.go](utils.go)) unwraps only data-mongo collections and returns nil for anything else, and [queries.go](queries.go) names this package as the seam to rewrite if the database ever changes.

## v027 keys ACTOR rules by the RAW trigger, so both MatchKey shapes live on disk

`Rule.Save` keys ACTOR rules by the resolved actor URL, but [upgrades/v027.go](upgrades/v027.go) computes `model.RuleMatchKey` from the raw stored trigger, because a migration cannot resolve handles over the network. Both shapes persist, and point lookups probe both (`loadActorRule` in [../service/rule_blocks.go](../service/rule_blocks.go)). Never write a migration that "fixes" raw-keyed rules by resolving them, since that puts network calls in a migration. [../model/AGENTS.md](../model/AGENTS.md) covers the matching side.

## Nothing purges SearchResults by age

No task queued by [consumer/scheduleDaily.go](../consumer/scheduleDaily.go), [scheduleHourly.go](../consumer/scheduleHourly.go), or [scheduleStartup.go](../consumer/scheduleStartup.go) purges SearchResults. `queries.Recycle` removes only rows whose `deleteDate` is more than 30 days old, SearchResults are hard deleted and never carry one, and the collection has no TTL index. A SearchResult lives until something deletes it explicitly, so assume no retention window when reasoning about growth or stale rows.

## An aggregation-pipeline `$set` BROADCASTS across an existing array

An update pipeline whose new value references the field it replaces (`{$set: {loc: {type: "Point", coordinates: "$loc"}}}`) does **not** replace an array-valued field: it produces an array of N identical documents, one per original element. Only a real MongoDB shows this. Use a plain, non-pipeline `$set` with a literal document computed in Go.

## Three search indexes key on fields nothing writes

`model.SearchQuery` has neither `notifiedDate` nor `timeoutDate`, so `idx_SearchQuery_Notified`'s partial filter (`notifiedDate: 0`) matches no document and the index is empty. `idx_SearchResult_Notified` and `idx_SearchResult_Locked` index every SearchResult for `notifiedDate`, `timeoutDate`, and `lockId`, which nothing has read or written since 2025-11-06; `queries.LockSearchResults`, which wrote the last two, has no caller. The allow-list in `sync/searchResult_test.go` hides all three keys. [BUG-332](../../emissary-specs/bugs/BUG-332-Three-Search-Indexes-Key-On-Fields-Nothing-Writes.md) removes them.

## A Stream's rank starts at 1, and 0 means "not set"

`build.StepSort` never assigns 0, and `Stream.Save` gives any Stream ranked 0 the rank `MaxRank` returns: one after its highest-ranked sibling, or 1 for a parent with no children. Bandwagon publishes `rank` as a song's `position` and ID3 track number, so a first child ranked 0 federated as track 0 and moved after its siblings on its next save. Keep ranks counting from 1 (`TestMaxRank`).

## The hourly unread scan depends on a partial index, and on sending both of its terms

`NotificationUsersWithUnread` asks for every User holding an old unread notification, so it cannot use the indexes that lead with `userId`. `idx_Notification_UnreadByDate` serves it: `createDate` then `userId`, holding only rows where `deleteDate` is 0 and `readDate` is `math.MaxInt64`. MongoDB uses a partial index only when the query names the index's own filter terms, so the query must send both as equalities. Drop either one and the scan reads the whole collection, with no error. `TestNotificationIndex_UnreadByDate` in [sync](sync/notification_mongo_test.go) runs `explain` on the same filter.
