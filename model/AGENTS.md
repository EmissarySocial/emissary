# model — Notes for AI Agents

Emissary's domain data structures — see [README.md](README.md) for the package's shape and [doc.go](doc.go) for the file-role conventions (struct + constructor, `_accessors`, `_activitypub`/`_jsonld`, `_constants`). Models carry data and vocabulary only; loading, validation calls, and side effects live in [service](../service/AGENTS.md). Repo-wide rules are in [../AGENTS.md](../AGENTS.md).

## The ID field maps to `bson:"_id"` and the constructor mints it

Every persisted model tags its ID field `bson:"_id"` (no `,omitempty`) and its `NewXxx()` constructor sets `primitive.NewObjectID()`. Get either half wrong and nothing errors: with a differently-named tag, Mongo mints its own `_id` on insert while the struct's ID loads back zero, so every later update/delete filters `_id == 000…0` and matches nothing; without the minted ID, zero `_id`s collide across inserts. `IsNew()` keys off `journal.CreateDate`, not the ID, so pre-minting the ID does not break creation.

## `Fields()` projections are unchecked strings — pin them in [fields_test.go](fields_test.go)

A projected name that matches no bson tag asks Mongo for a field that cannot exist, and the field it was meant to name silently loads as its zero value. `TestFieldProjections` verifies every `Fields()` list against the struct's bson tags; add each new model or summary type to its table. `RuleSummaryFields()` shows the stakes: dropping `userId` there would make the disposition engine read every rule as ADMIN-tier.

## Renaming a bson field requires a migration in [../queries/upgrades](../queries/upgrades/)

Legacy rows keep their value under the old key while the renamed field reads back zero — no error surfaces until a lookup misses or a unique index refuses to build. Collection's `collectionType` field (formerly `type`) is the precedent: the rename shipped without a migration and stalled boot-time index builds until an upgrade slot reconciled it. See [../queries/AGENTS.md](../queries/AGENTS.md).

## `Stream.IsMyself` always returns false — it is not an author check

`IsMyself` (part of the `AccessLister` interface) means "does this object directly represent this User's own profile", which a Stream never does. The author predicate is `stream.IsAuthor(userID)`, which also rejects a zero ID so an anonymous request can never match a zero-author stream. Using `IsMyself` as an author gate silently rejects the real author too — see [../handler/mastodon/AGENTS.md](../handler/mastodon/AGENTS.md).

## `Rule.MatchKey` is the identity and lookup key; `Trigger` is display-only

`MatchKey` (`"<TYPE>:<normalized trigger>"`) is computed in service `Rule.Save` — for ACTOR rules from the resolved canonical actor URL, while `Trigger` keeps the friendly form the user typed (possibly a webfinger handle). It is deliberately absent from `RuleSchema` and `GetPointer` so no form can set it; keep it that way. Two key shapes exist on disk: rules saved through `Rule.Save` key ACTOR rules by canonical URL, but rules backfilled by [../queries/upgrades/v027.go](../queries/upgrades/v027.go) key by the raw trigger (a migration cannot resolve handles over the network), so a point lookup by actor must probe both shapes — `loadActorRule` in [../service/rule_blocks.go](../service/rule_blocks.go) does this on purpose; do not "simplify" the double probe.

## `RuleMatchKey` and `DocumentMatchKeys` normalizers must stay paired

Whatever normalization runs on the rule side in [ruleMatchKey.go](ruleMatchKey.go) must also run on the document side, or the two key sets stop intersecting and rules fail OPEN — blocked content passes with no error anywhere. `normalizeActorURI` returns non-URL input trimmed but otherwise unchanged, which is why `Rule.Save` must resolve handles to URLs first: a handle-keyed ACTOR rule can never match a document, whose keys always come from actor URLs. `ActorMatchKeys` excludes TAG keys by design (the wire gate filters by WHO, never by content) — see [../service/AGENTS.md](../service/AGENTS.md) for the enforcement-surface consequences.

## Hashtag URLs are absolute and built only by [hashtag.go](hashtag.go)

Federated documents are read on other servers, where a relative href cannot resolve, so `HashtagURL`/`HashtagURLPrefix` anchor the Template's path-prefix `tagUrl` (`/search?q=`) to the current hostname at generation time — the Template never stores a hostname because one server hosts many domains from one template set. An empty `tagUrl` means "extract but do not linkify". The tag escaping matches `replace.Linkify` exactly so an AP `tag[].href` equals the anchor written into the content it describes. Never concatenate a hashtag URL by hand at a call site.

## Schema strings default to no-html; verbatim fields need `unsafe-any`

A `schema.String` with no `Format` strips tags and collapses whitespace on the write path, which silently corrupts secrets, tokens, and scopes — vault maps, refresh-token hashes, and OAuth scopes all use `Format: "unsafe-any"` with a `MaxLength` for exactly this reason. Federated-ingest fields (AP IDs, media types) get a length bound but NO token/url format, because valid remote values (`application/ld+json`, `tag:` URIs, colon-delimited scopes) contain characters those formats reject, and rejecting on ingest drops the record. `"url"` requires http/https plus a host; `"uri"` is looser (any scheme, no host required); `"webfinger"` strips the leading `@`, so a webfinger-format field cannot round-trip through a schema Set.
