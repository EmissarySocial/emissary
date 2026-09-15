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

## `Follower.Data["secret"]` is plaintext on purpose

It is the token in an EMAIL Follower's own confirmation and unsubscribe links, and that is all it authorizes: the worst a leaked one allows is unsubscribing that one address. It is not a credential and does not belong in a `Vault` — `UserConnection` and `MerchantAccount` encrypt theirs because a leaked API key reaches the User's whole third-party account. Do not flag it, and do not move it. The older readers spell the key as the literal `"secret"` rather than `FollowerDataSecret`; that is fine too — the key is the contract, not the spelling — so leave them.

## A schema `Enum` does not reject the empty string — a state field must also be `Required`

rosetta's string validator checks the enum only when the value is non-empty (`(stringValue != "") && !Contains(...)`), so `schema.String{Enum: [...]}` silently admits `""`. That is right for an optional field and wrong for a state field, where an empty value is *no state* — it matches none of the predicates and the record renders as nothing. `UserConnection.status` carries `Required: true` for exactly this reason, and `TestUserConnection_EveryStatusValidates` pins both halves: every named state validates, and `""` does not. The mirror-image trap shipped the same day: adding a new state to the constants, the constructor, and every assignment site while forgetting the enum, which refused every new record with "Must be one of the specified values" — so a whole-record `Validate` in every state is the test to write when a state is added.

## Schema strings default to no-html; verbatim fields need `unsafe-any`

A `schema.String` with no `Format` strips tags and collapses whitespace on the write path, which silently corrupts secrets, tokens, and scopes — vault maps, refresh-token hashes, and OAuth scopes all use `Format: "unsafe-any"` with a `MaxLength` for exactly this reason. Federated-ingest fields (AP IDs, media types) get a length bound but NO token/url format, because valid remote values (`application/ld+json`, `tag:` URIs, colon-delimited scopes) contain characters those formats reject, and rejecting on ingest drops the record. `"url"` requires http/https plus a host; `"uri"` is looser (any scheme, no host required); `"webfinger"` strips the leading `@`, so a webfinger-format field cannot round-trip through a schema Set.

## A `Vault` nonce is an OUTPUT of sealing, never an input

`Vault` carries `Nonces mapof.String`, one per value, and `Encrypt` mints a fresh nonce on every seal. The invariant is **per-encryption, not per-key**: a nonce that is stored and reused when a value is *updated* still collides with whatever older ciphertext an attacker already holds. AES-GCM under a repeated (key, nonce) pair leaks `C1 ⊕ C2 = P1 ⊕ P2` and permits GMAC subkey recovery, and the live instance of this was a Stripe Connect vault holding a publishable key, a restricted key, and a webhook secret together — two of them with known `pk_live_`/`rk_live_` prefixes.

Three things to keep. Go's GCM **panics** rather than errors on a wrong-length nonce, so anything read from the database is length-checked in `nonceFor` before it reaches `Open`. `Decrypt` falls back to the legacy single `Nonce` field for records not yet re-saved; the scheme is additive and self-migrating, with no upgrade slot, so do not remove the fallback. And re-saving an unchanged secret now produces different ciphertext every time, where the old format produced identical bytes — a diff between two saves is not evidence that the secret changed.

## `Theme.Datasets` and `Template.Datasets` describe one concept and only one of them worked

`Template.Datasets` is a `DatasetMap` (`map[string]form.ReadOnlyLookupGroup`) — an **array** of `form.LookupCode` per dataset, which is what `options:{provider:"x.y"}` resolves against in `service/lookupProvider.go`. `Theme.Datasets` was a plain `mapof.Object[mapof.Any]`, a **map**, so the array spelling a theme author would naturally write parsed into nothing and the dataset was silently inert. It is now the same `DatasetMap` type. Dataset providers remain **Template-only**; a Theme cannot supply one.

## A bson-only struct opts EVERY field into `json.Marshal`

The absence of a `json` tag is not a decision — it is the default, and the default is "export it". `GET /@:userId/export/:collection/:recordId` marshals model objects straight to JSON, and because `model.User` carried only `bson` tags, that route emitted `User.Password` (the hash) and `User.PasswordReset` (a live account-takeover code). The route requires an OAuth grant — `WithOAuthUser` rejects a plain session — so a third-party app authorized for portability was the *only* caller, and it received both. They now carry `json:"-"`. **Any new secret-bearing field on a model needs an explicit `json:"-"`**, and any new route that marshals a model wholesale needs to be checked against this.

The mirror-image rule is that **not every secret-shaped field is one to suppress**. The export's consumer is the server a User is migrating TO, which needs their records whole: `Following.Secret` and `MerchantAccount.Plaintext` travel on purpose, and `MerchantAccount.ExportDocument` goes further and decrypts the vault into an explicit map. Decide by asking whether the destination server needs the value, not by the field's name. `service/export_credentials_test.go` pins both directions, so a new `json:"-"` on an allowed field fails there rather than silently breaking a migration.

## `model.DetectContentType` may use the filename to disambiguate, never to promote

It extends `http.DetectContentType`, which cannot sniff FLAC, M4A, Ogg audio, bare MP3, ADTS AAC, WMA, or AMR — all of which came back as `application/octet-stream` and failed an `audio/*` upload gate. The contract is that **the filename argument can never promote bytes into a media type**; it only picks audio-versus-video *inside a byte-confirmed container* (EBML `.weba`, ASF `.wma`/`.wmv`, ISO-BMFF `.m4a`/`.m4b`/`.3ga`). Byte-first sniffing stays the security boundary, because the serve-side inline decision is built on it. `TestDetectContentType_FilenameCannotPromote` pins this; do not relax it into a filename fallback. Known edge: `CanServeInline` also needs `mime.TypeByExtension` to recognize the extension, and `.amr`/`.3ga` return empty, so those download rather than playing inline.
