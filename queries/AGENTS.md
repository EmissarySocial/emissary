# queries — Notes for AI Agents

Custom MongoDB queries that don't fit the [service](../service/AGENTS.md) layer's standard CRUD — see [README.md](README.md). [upgrades](upgrades/) holds the per-version data migrations, [sync](sync/) holds index definitions plus the reconcile passes that make them buildable. Repo-wide rules, including the upgrade-slot rule this package enforces, are in [../AGENTS.md](../AGENTS.md).

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
