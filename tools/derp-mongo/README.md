# derp-mongo

Emissary's runtime errors go to MongoDB through this package. It implements `derp.Reporter`, so every `derp.Report` call anywhere in the server becomes one `ErrorLog` record. [server/factory_core.go](../../server/factory_core.go) is the only caller, wiring it up when `config.json` names a logger of type `mongo`.

Reading those records back is a separate program, [benpate/derp-triage](https://github.com/benpate/derp-triage), which turns the collection into a work queue.

## What a record carries

Four of the fields are the error itself: the HTTP status code, the root location and message, and the whole nested `derp` chain. The other four exist so that a log can be worked through rather than merely read.

| Field | What it holds |
| --- | --- |
| `signature` | The stable identity of one defect, shared by every occurrence of it |
| `status` | `new`, `fixed`, or `ignored` |
| `statusNote` | Why that decision was made |
| `statusDate` | When it was made |

`newRecord` derives the signature from the very values it stores beside it, so a record can never describe one error while its signature describes another.

## Why the signature is computed here

It is computed at report time, from the live error, and never rebuilt from the stored document. That is not an optimization — the reconstruction is impossible. A plain Go error with no location anywhere in its chain marshals to an empty BSON document, so there is nothing left to hash: two errors with entirely different messages produce the same identity from their stored records, and different identities from the live ones.

The algorithm lives in Emissary rather than in `benpate/derp` because its normalization rules are Emissary-shaped. `variableObjectID` matches a MongoDB ObjectID, and `variableHostname` is aggressive enough to rewrite `config.json` as a host. Exporting those would make them a public contract for every `derp` consumer, and `derp`'s semver would become a silent migration vector where a routine dependency bump re-partitions somebody's queue with no diff in Emissary at all.

## The version prefix is not decoration

A signature renders as `v1:` plus twelve hexadecimal characters. Any change to the normalization rules, the seed layout, the separator, or the truncation length **must** bump it.

Triage decisions are stored beside the signature and keyed by it, so an unversioned change would split a decided signature from its own future occurrences. Every error somebody had already fixed would come back as new work, and nothing would report that anything had happened, because a signature matching nothing looks exactly like a signature that was never decided. The prefix makes that split legible instead of silent.

`normalizeMessage` is single-pass and deliberately not idempotent; see [../AGENTS.md](../AGENTS.md) for why a second pass is wrong and why word-safe placeholders do not fix it.

## A decision lives exactly as long as its records

[consumer.PurgeErrors](../../consumer/purgeErrors.go) deletes everything older than seven days, by date alone, with no regard for status. That is deliberate and this package does nothing to change it.

The consequence is worth stating, because it looks like a bug and is not. Triage reads a decision across every record sharing a signature, so once the last marked record ages out the decision is gone with it and the error returns to the queue. A `fixed` or `ignored` defect therefore stays quiet for about a week, not forever.

Keeping a decision longer was designed and rejected: it means exempting decided records from the purge and pruning them to one tombstone per signature, which is real machinery in the retention path to buy a week. Do not add it back without asking — triage decisions are meant to expire here.

## Tests

`integration_test.go` and `purge_test.go` write to a real MongoDB, skip under `-short`, and skip rather than fail when no database answers. Each one gets a throwaway database named for a fresh ObjectID. Local connect strings need `?directConnection=true`; see the root [AGENTS.md](../../AGENTS.md).
