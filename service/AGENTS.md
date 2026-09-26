# service — Agent Notes

See [README.md](README.md) for what this package is. These are the non-obvious rules.

## Never hand `golang.org/x/oauth2` the default HTTP client

`oauth2.Config.Exchange` and `TokenSource(...).Token()` POST to a **remote-actor-declared** `TokenURL`. With no client in the context they fall back to `http.DefaultClient`, which is UNGUARDED — a blind POST SSRF that reaches internal hosts. Always pass a context carrying `remote.NewHTTPClient(...)` under the `oauth2.HTTPClient` key. Both entry points do this through an `oauthHTTPContext` helper: [import_oauth.go](import_oauth.go) (source-actor endpoints — attacker-controlled) and [domain.go](domain.go) (provider endpoints — admin-configured, defense-in-depth). Pass `activityService.AllowPrivateIPs()` so local/self-federation dev still works.

## Import fetches must be same-origin with the source actor

The user's OAuth Bearer token is scoped to the **source server**, and migration collections legitimately reference third-party hosts (a following/blocked list, a boosted post). So every import dereference is gated with `uri.NotSameOrigin(url, sourceOrigin)`: `consumer/importStartup.go` skips off-origin documents at item creation, `Import.ImportAttachments` skips off-origin attachments, and `consumer/importItems.go` re-checks at the fetch sink as belt-and-suspenders. `doAuthorize` likewise origin-pins the OAuth `AuthURL`/`TokenURL` to the actor.

This is about **credential leakage and content-injection, not private-IP SSRF** — a raw `remote.Get` already blocks private IPs by default (see [remote/AGENTS.md](../../../benpate/remote/AGENTS.md)). The residual risk the same-origin gate closes is the token traveling off-origin and foreign content being saved as the user's record.

## `AllowPrivateIPs` comes from the ActivityStream service

`activityService.AllowPrivateIPs()` is the one predicate for "may this instance talk to private addresses" (true only on a local/private hostname). Thread it into any guarded-client construction here rather than re-deriving it.

## A `UserConnection` is a User's own credential for a third-party service — six rules a second provider will need

`UserConnection` holds one row per `(UserID, Type)` with an `IsActive delta.Bool`, a `Status`, a `Data mapof.String` for non-secret remote IDs, and a `Vault` for secrets. Mailchimp is the only type today; `MerchantAccount` has the same shape and has not migrated. What follows is not visible from the code.

**`connect()` runs BEFORE `collection.Save`, not after.** A credential the remote service refuses must never reach the database, because everything downstream reads a stored connection as a working one. (`connectBluesky` in the Following service runs the other way round; do not copy that here.)

**`IsActive` is a `delta.Bool`, and its CHANGE is the trigger.** Write it through the schema — `SetBool` calls `IsActive.Set` — never by assigning the field, or the change is lost and the connect/disconnect never fires. `Status` is deliberately separate: `IsActive` is what the User asked for, `Status` is whether it works, and folding a rejected credential into the switch would make Emissary look like it turned the connection off.

**`IsReady()` means "switched on AND finished being set up".** It tests `Status == READY`, not `!= RECONNECT`, because a connection whose credential works but whose setup never completed is still `PENDING` (D48) — a real value, assigned by the constructor, never the empty string. Since D47, "set up" includes the webhook on a public domain and deliberately excludes it on a local one (D45) — that decision lives in `mailchimp_setup`'s ordering, not in the predicate, because the model has no host to ask. Every consumer — the inbound webhook, the outbound sync — leans on that distinction.

**There is deliberately no `Fields()` projection, and a test asserts its absence.** `QueryBuilder` projects with `T.Fields()`, and `MerchantAccount`'s projection omits `vault` — which here would make a configured connection read as unconfigured, and compare a presented webhook secret against an empty string.

**The record is HARD deleted, and is not `Importable`.** It holds a live credential, so a soft-deleted copy keeps that credential readable by anything querying without `notDeleted`; hard delete is also what lets `{userId, type}` be a plain unique index instead of a partial one. And an account import that installed someone else's API key would hand this server a credential it should not have.

**A failed remote teardown is reported, never propagated.** Both `Delete` and the pause path do this: a User whose key was already revoked at the far end must still be able to disconnect, and refusing would strand them with a connection they cannot switch off.

Two further notes on the write paths. `Save` is reached *only* from the settings form, so there are no background writes to keep off the network — which is why an active connection is re-proved on every save, and why that re-run is what restores anything the User deleted by hand at the far end. And a queue task that discovers a rejected credential must write `Status` through `collection.Save` directly, **not** through `UserConnection.Save`, which would call `connect()` and reach for the same credential that was just refused.

## A Follower's email address is stored lowercased, and every lookup normalizes to match

`Follower.Save` is the one writer and it applies `model.NormalizeEmailAddress` (trim + lowercase) to `Actor.EmailAddress` — and to `Actor.ProfileURL` **only when `Method == EMAIL`**, because for everyone else `ProfileURL` is a real URL whose path is case-sensitive. `LoadByEmailAddress` normalizes its argument; `LoadByActor` does not (it serves ActivityPub too), so a caller passing an email to it — `handler.PostEmailFollower` through `LoadOrCreate` — normalizes first. Mailchimp hashes members on the lowercased address, so an inbound webhook naming `Sarah@Connor.MIL` must find the row stored as `sarah@connor.mil`; before this rule it silently did not. Rows written before the rule are brought in line by upgrade slot 30.

`Follower.Data["secret"]` — the unsubscribe-link token that `LoadBySecret` checks — is plaintext by design; see [model/AGENTS.md](../model/AGENTS.md).

## `User.GetJSONLD()` output is fingerprinted — keep it deterministic

Every `User.Save` hashes `GetJSONLD()` into `ProfileFingerprint`; a changed hash federates an ActivityPub `Update` to all followers ([user.go](user.go) Save, [user_activitypub.go](user_activitypub.go) `sendProfileUpdate`, spec PROFILE-UPDATE-FEDERATION.md). Adding anything volatile or per-save (timestamps, counters, random values) to `User.GetJSONLD()` makes every save — including signin bookkeeping — spam followers with Updates. `TestUser_CalcProfileFingerprint` pins which fields participate; update it when the actor document gains a field.

## TAG rules only exist in the full-document key set

`model.ActorMatchKeys` deliberately excludes content (TAG) keys — it answers "is this actor filtered?", nothing more. Any enforcement surface that should honor TAG rules (newsfeed ingest, notifications, render labels) must evaluate `model.DocumentMatchKeys` / `Rule.Disposition` on the (unwrapped) payload, or TAG rules silently never fire there. A surface built on `ActorDisposition` alone looks complete and passes every identity-rule test while ignoring hashtag rules entirely.

## A boolean in a Theme's `themeData` needs an explicit `default`, or its first save flips it

A Theme's hjson schema layers `themeData` properties onto the Domain's own wildcard object ([build/builder_admin_domain.go](../build/builder_admin_domain.go) `schema()`), and the settings form is a tab layout — so saving **any** tab posts **every** field. `model.Domain.ThemeData` starts out an empty `mapof.Any`, and an absent boolean reads back as `false`. A toggle meant to start ON therefore renders OFF on a pristine Domain, and the owner's first save of an unrelated tab writes that phantom `false` — silently blanking whatever the flag controls.

Declaring `default:true` on the property is what fixes it, and **both** halves read that one declaration: `form/widget.Toggle` falls back to the schema element's `DefaultValue()` when the object carries no value, and [build/builder_common.go](../build/builder_common.go) `ThemeData` falls back to `theme.Schema.GetElement("themeData." + token).DefaultValue()`. Keep them in step — a default that only one side honors is worse than none, because the page and the form that configures it then disagree. `ThemeData` returns a **string**, so compare against `"true"`, never truthiness.

Note that `schema.Boolean.Default` does *not* surface through `schema.Schema.Get`: `getProperty_Boolean` errors on a missing map key rather than falling through to the default, which is why both fallbacks are written out by hand instead of coming for free.

## The email library sanitizes header VALUES, not header NAMES

Outbound mail goes through `serverEmail.go` to `github.com/xhit/go-simple-mail/v2` v2.16.0. Two things about that boundary are invisible from Emissary's own code. Header **values** are already CRLF-safe — the library escapes them — so do **not** add a second sanitizer of our own. Header **names** are not checked, so a name assembled from data is the injection surface, not the value.

The other trap is a template one: a Go template that renders an optional header emits the literal string `<no value>` for a missing key rather than nothing, which produces a malformed header instead of an absent one. Guard the whole header line with `{{if}}`, not just its value.

When reading this dependency, confirm which copy you have: the module cache also holds a stale `go-simple-mail@v2.2.2+incompatible` tree under a different module path. Use `go list -m -f '{{.Dir}}' github.com/xhit/go-simple-mail/v2`.

## A template reload overwrites its libraries, and nothing may empty one

`Template.loadTemplates` fills five libraries from the template folders: templates, themes, widgets, registrations, and server emails. Each `Add` writes a definition over its live copy, and no library is ever reset. Keep it that way. A reset in one place and a refill in another fails silently whenever the refill is skipped. `ServerEmail.Refresh` emptied the email library on every config reload, while `Template.Refresh` skipped the refill whenever the template locations were unchanged, which is almost always. Registration then failed until a restart ([BUG-180](../../emissary-specs/bugs/_done/BUG-180-Config-Reload-Empties-The-Server-Email-Registry.md)). The cost of overwriting is that a definition deleted from disk stays loaded until a restart; that gap is deferred.

Two locks keep a reload safe. `Template.reloadLock` lets one load run at a time, because a file change and a config reload can each start one, and every load writes the shared prep area and the other four services. Each library also has its own lock, and every read outside a load takes it, even a `len`. A load writes these maps while requests read them, and a concurrent map read and write in Go is a fatal error that `recover` cannot catch. Code that runs inside a load, such as `Theme.calculateAllInheritance`, is already serialized by `reloadLock` and reads without it.

## The cached Domain record is read-only, and a writer loads its own copy from the database

`Domain.Cached()` returns a `*model.Domain`, a pointer into the snapshot held in the service's `cache` field, an `atomic.Pointer`. It costs one atomic read and nothing else: `service.cache.Load()` is that read, and `&result.Domain` is the address of the embedded record, not a copy. The database read is `service.Load(session, &writableDomain)`. `Save`, `Start`, and `queries.WatchDomain` replace that snapshot through `publish`, which is how a Domain saved on one server reaches the others ([BUG-170](../../emissary-specs/bugs/_done/BUG-170-Domain-Record-Cached-Per-Node.md)). A `model.Domain` cannot be saved: the journal that `data.Object` needs lives on `model.WritableDomain`, and `Save` takes only that. To change the record, check the cheap thing on `Cached()`, then `Load` → edit → `Save`: build the value with `model.NewWritableDomain()` and `Load(session, &writableDomain)` decodes the stored record into it, so the writer starts from the database, owns every map and slice, and inside a transaction sees the earlier writes of the same request. `Load` does not reset its target, the same as every other service's `Load`; the decoder merges into maps the target already holds ([../model/AGENTS.md](../model/AGENTS.md)), so always hand it a fresh constructor value. After `Load`, re-check what the cached record showed, because another request or node may have got there first; `PrivateKey`, `vapidKeys`, and `stampHostname` all do. The design and its decisions are in [DOMAIN-READ-WRITE-SPLIT](../../emissary-specs/projects/_done/DOMAIN-READ-WRITE-SPLIT.md).

Two rules remain by convention. Never write through the pointer `Cached()` returns, even into one of its maps: the type system stops a `Save`, not an assignment, and every request shares that record ([BUG-175](../../emissary-specs/bugs/BUG-175-Domain-Get-Returns-A-Writable-Cache-Pointer.md)). And never keep that pointer beyond one call, because the next publish replaces it; this is why the Connection service reads through `Cached()` every time. `publish` stores a `Clone()` of what it is handed, so a writer that keeps editing after `Save` (a builder's pipeline runs on past its `save` step) never reaches the cache. `bootstrap` is the one writer that does not `Load`, because no record exists yet, and `UpgradeMongoDB` writes `databaseVersion` with `$set` and never touches the cache, which is why `WatchDomain` opens with `UpdateLookup`.

`Save` is still a blind whole-document replace, so a server whose loaded copy is stale can write it over a newer one; `Load` narrows that window to the time between a writer's `Load` and its `Save`, and the watcher then republishes within milliseconds. `Save` also publishes before the request's transaction commits ([BUG-176](../../emissary-specs/bugs/BUG-176-Domain-Save-Publishes-Before-Commit.md)). Without change streams (a standalone MongoDB), servers never see each other's Domain changes, so a cluster must run on a replica set.

`Domain.Refresh` must never reset the snapshot. It runs on every configuration reload, but only `Start` and the watcher reload the record, so a blank published there strands an empty Domain with no `Label`, `PrivateKey`, or `CreateDate`, and its next `Save` tries to INSERT a second record. `Start` resets it next to the `Load` that refills it, and a failed `Load` leaves that blank published.

**Tests that need a real decode run against the local replica set.** `data-mock` stores the saver's own pointer and hands `Load` shallow copies of it ([BUG-178](../../emissary-specs/bugs/BUG-178-Data-Mock-Shares-Memory-With-Stored-Records.md)), so under the mock a loaded record shares its maps with the "database", and a rejected save can still change what is stored. `newReplicaSetSession` in `domain_test.go` skips when no replica set is reachable; use it for anything that asserts a loaded record is independent, that a failed save left the database alone, or that a transaction reads its own write.

## A Connection handed out by the Connection service must not share maps with the snapshot

`Connection.Load`, and every `Load*` method built on it, clones `Data`, `Vault.Encrypted`, and `Vault.Nonces` before returning, because its callers write into them: the `edit-connection` step through the settings form, `NewOAuthClient`, `Vault.Encrypt` inside `Save`, and `StripeConnect.Connect`. Without the clone, a `Save` that fails leaves its rejected edits in the published record, and the next Domain save from that server stores them. The vault's unexported `plaintext` map is still shared, because only `model` can copy it. `Query`, `QueryAll`, `ActiveByType`, and `AllAsMap` return shared values, so treat them as read-only.

**`Connection.Save` seals the vault only after `provider.Connect` runs, and must keep that order.** `Connect` can add a secret of its own: `StripeConnect` writes the webhook signing secret that Stripe returns when it creates the endpoint. Sealed first, that secret stayed in the vault's `bson:"-"` plaintext map and was never stored, so every live Stripe Connect webhook failed verification ([BUG-179](../../emissary-specs/bugs/_done/BUG-179-Stripe-Connect-Webhook-Secret-Never-Stored.md)). `MerchantAccount.Save` still seals before it connects; that path is the disabled direct-Stripe provider, and restoring it needs the same fix.

## A lost Stripe webhook secret is replaced, never recovered, and the replacement costs events

Stripe returns an endpoint's signing secret only in the response that creates it, so `StripeConnect.Connect` replaces any endpoint whose secret is missing: it creates the new one, then deletes the old one. Deleting an endpoint also cancels Stripe's retries of every event it failed, and those events are the only record of subscriptions that ended. So a repair is always followed by `MerchantAccount.ReconcileStripeSubscriptions`, which asks Stripe for each `sub_` Privilege's status and revokes the ones that have ended. `consumer.ScheduleStartup` runs the repair on every boot, which makes it a no-op once each connection holds its secret.

The reconciliation deliberately runs outside a transaction. It makes one Stripe call per subscription, which would soon outlast MongoDB's 60-second limit on a transaction, and each revocation stands on its own.

## Domain bootstrap is one transaction, and the invariant is what matters

`Domain.Start()` delegates to `bootstrap(session, &domain)`, which fills the record `Start` built, and wraps its write and `createOwner` in a single transaction; `Start` publishes that record only once `bootstrap` has returned, through the same `publish` a loaded record goes through. That establishes **domain record exists if and only if an owner exists** (when `CreateOwner` is set), so a failed first boot writes nothing and the next boot retries cleanly. Before this, the owner was a separate non-transactional write gated on "domain record not found": any failure stranded a domain record with no owner and the gate never re-ran, which locked every demo and fresh instance out. `persist()` is the write-only path used inside the transaction; the in-memory domain cache is published only **after** commit. Do not collapse `persist` back into `Save`.

Three decisions here look like defects and are not. The `demo` default password is set **only** under `IsLocalhost()`; that gate is the thing keeping a known credential off public hosts. `config.Owner` has no password field by design — a non-localhost owner signs in first through an emailed reset link, or the operator gets a loud warning pointing at the setup console. And `newOwnerFromConfig` falls back from a blank email to `demo@<hostname>` because `User.Save` requires a non-empty address.

## The server configuration owns the hostname, and the Domain record keeps a stamped copy

`Domain.Host()` builds every derived URL from the record's own `Hostname` (the federation actor, OAuth client metadata, oEmbed, email links), so the record needs a copy even though an operator can rename a domain in the setup tool at any time. `bootstrap` stamps it before the first write, and `stampHostname` checks it on every `Start` and rewrites it when the two differ, which is why `shouldStartDomainService` restarts the service on a rename as well as on a reconnect. Two guards keep this safe. A blank configured hostname never overwrites a stored one (`needsHostnameStamp`), because the setup console builds factories before its configuration is complete, and a cleared hostname breaks every derived URL. And `Start` never runs before a database is configured, because it needs a session.

## `Factory.Steranko` is the only place the password hashing policy is set

Every password write goes through the Steranko instance it builds. A path that sets a password any other way stores it raw (CWE-256), which is how registration, password reset, and Mastodon signup went wrong; `TestSteranko_SetPassword_StoresBCrypt12` pins the policy. New hashes use BCrypt cost 12, about 200ms each: slow enough to resist offline cracking, and fast enough that signin latency and a failed-signin flood stay affordable. The Plaintext fallback lets passwords stored before hashing still sign in, and steranko re-hashes them on first use. When the plaintext-password migration ships, remove the fallback and delete `TestSteranko_PlaintextFallback` deliberately.

## A captured server-level handle fails with symptoms that point elsewhere

`Factory` reads the common database, the task queue, and the server email service through `serverFactory` on every call, as [server/AGENTS.md](../server/AGENTS.md) requires, because a config reload can rebuild all three. A copy captured anywhere in this package goes stale on that reload. A stale common database fails every call with `client is disconnected`, which surfaced as inbound signature verification failing with a bare 401, and a stale queue silently drops every task with `Turbine Queue: stopped`. [factory_lifecycle_test.go](factory_lifecycle_test.go) pins the read-through.

## `Factory.Close` releases what one domain owns, and nothing the server shares

A domain factory owns three things that never stop on their own: its change stream watchers, its realtime broker's goroutine, and its MongoDB client, whose connection pool stays open until it is disconnected. `Close` releases all three, and the server calls it whenever a factory leaves the registry (see [../server/AGENTS.md](../server/AGENTS.md)). `NewFactory` closes a factory whose first `Refresh` fails, and a reconnect disconnects the client it replaced once the new watchers are running.

Three rules keep it safe. **Never close a service the server passed in:** everything `server.refreshDomain` hands `NewFactory` by pointer (the JWT, content, registration, template, theme, and widget services, and the HTTP cache) is one instance shared by every domain, and closing the JWT service from one domain empties the key cache that every domain signs with. **Never close `sseUpdateChannel`:** a request still in progress on a dropped factory may send on it, and a send on a closed channel panics. **The disconnect runs in the background:** `Close` is called while the server holds `reloadLock`, and requests already in progress get up to 30 seconds to finish, while a query that starts after the disconnect fails.

## Two geocoder response mappers are wrong, and the tests pin the bug

Both are asserted as current behavior with "asserting the BUG" comments rather than fixed, so a fix will surface as a test failure. `mapGoogleSearchResult` matches `administrative_level_1` while Google actually sends `administrative_area_level_1`, so `Address.Region` is silently empty for **every** Google geocode. And `mapMaptilerAddress` builds `Street1` by concatenating house number and text, which yields a bare space when both are empty. The suite is non-network by construction; the methods that do reach the network need URL injection before they can be covered at all.

## An Identity is bound to the actor whose inbox received the guest code, never to a handle

A guest Identity carries Privileges, and Privileges are keyed by actor URL. `Identity.Save` then hands the Identity every Privilege keyed on its `ActivityPubActor` (`uniquify` and `Privilege.refreshIdentity`), so whatever writes that field is the whole authentication. Before BUG-01 the guest code was delivered to whatever a typed handle resolved to at *send* time, while the confirm handler stored the handle and `Save` re-resolved it a *second* time through a bare `digit.Lookup`. Anyone controlling a domain could receive the code, repoint their WebFinger record at someone else's actor, click the link, and inherit that actor's Privileges. No WebFinger verification closes that, because the same domain owner answers both lookups.

The rule now is that `ActivityPubActor` is written only from an actor the server resolved itself, and the code carries that resolved id (`SendGuestCode` mints the JWT with `T = ACTIVITYPUB` and `A = actor.ID()`), so the actor whose inbox received the code is the actor the confirm handler binds. Four things follow. `LoadOrCreate` resolves a handle or URL to its actor **before** looking up, and looks up by actor id only, so two Identities can never share an actor. `WebfingerUsername` is always derived from the actor (`actor.UsernameOrID()`), never stored from input; `SetIdentifier(ACTIVITYPUB, …)` clears it so `Save` re-derives it. A vanity handle therefore displays as the actor host's handle (`@ben@emissary.social`, not `@ben@pate.org`) until a reverse-discovery loop-back exists, and that is accepted. `uniquify` strips the actor from every other Identity, which is safe only because the actor was proven by inbox delivery or resolved server-side for a Privilege grant. And `model.Identity.IsEmpty` counts the actor, because an actor with no `preferredUsername` leaves the handle empty, and `SaveOrDelete` used to delete such guests as empty.

Every actor lookup in this service goes through the package's `actorLoader` interface, satisfied by `ActivityStream.GetActor`. That is the cached client stack with the private-IP policy and the viewer's Rules; `digit.Lookup` must not be called from the Identity service. `RangeByIdentifiers` (here and in `Privilege`) drops blank identifiers before querying: an `exp.Equal("activityPubActor", "")` matches every record that lacks an actor, and `uniquify` used to walk most of the collection on every save.

## Stream tokens and usernames share the `acct:` namespace

A Stream whose Template declares an `actor:` block federates under a WebFinger handle, and that handle comes from one accessor, `model.Stream.ActivityPubUsername`: the token when it satisfies Mastodon's username grammar (ASCII letters, digits, underscores, with dots and dashes only between them), otherwise the StreamID. `StreamActor.JSONLD` writes it into `preferredUsername` and `Stream.WebFinger` writes it into the subject, so the two cannot drift. Before BUG-98 they were built from different fields, and the subject was a handle the server itself answered with 404, because `locateObjectFromAccount` returns `ActorTypeUser` for every `acct:` value that is not a reserved actor. Mastodon reconstructs `acct:<preferredUsername>@<host>` for every actor it fetches and rejects the actor when that lookup fails, so no Stream actor could be followed from Mastodon.

Three rules follow. `Locator.GetWebFingerResult` tries a handle as a User first and, only when that is a not-found error, as a Stream token; a User therefore shadows a Stream with the same handle, which is why `Stream.ValidateToken` refuses a token that matches a username and `User.ValidateUsername` refuses a username that matches a Stream token, both case-insensitively, because `LoadByUsername` ignores case. `Stream.WebFinger` answers 404 for a missing Stream and for a Stream whose Template has no actor, never 400, since a well-formed request for a resource this server does not publish is RFC 7033 §4.5's not-found case, and after the fall-through every unknown handle reaches it. And a token rename changes the handle: nothing keeps token history, so the old `acct:` form stops resolving at once, while the StreamID forms (`acct:<hex>@host` and the actor id `https://host/<hex>`, which `Save` always rewrites from the id) are the durable ones. Mastodon re-verifies the new handle on its next fetch and renames the account. Existing Streams whose token already matches a username are not migrated; they stay shadowed until renamed.

## An invalid signature refuses the request, and three rules keep that refusal honest

`resolveSignature` (in [permission.go](permission.go)) separates three cases an inbound request can present: no `Signature` header (Anonymous), a signature that verifies (an Actor), and a signature that fails to verify (a refusal). Collapsing the third into the first was BUG-20 — a peer with a misconfigured key got a normal-looking anonymous response, or a 403 naming the wrong cause, and went hunting a permissions bug that did not exist. `handler.resolveSignedActor` applies the same rule for the wrapper routes; the two are deliberate twins, so a change to one belongs in both.

**Nothing on the refusal path is logged or reported, and that is load-bearing.** The 401 reaches the peer, who is the only party who can fix a broken signature. A `derp.Report` would write the same event to the production error log that `derp-mongo` persists to MongoDB, where every misconfigured peer and drive-by probe would bury the reports that represent real defects. The objection is signal pollution, not volume. This works only because [server.go](../server.go)'s `errorHandler` special-cases unauthorized errors and every branch returns *before* `derp.Report` — so the refusal must stay an `Unauthorized`, and a refactor that gave it any other status would silently start filing peers into the error log. `TestResolveSignature_RefusalStaysOutOfDerp` is what catches that.

**The refusal message is fixed, and `err` never reaches the caller.** `errorHandler` writes `derp.Message(err)` into the response body, so anything the verifier said about *why* it failed would tell an unauthenticated prober which forgery attempt got closest. A local hostname is the one exception: it gets the `Mock-Key-Id` hint, because `errorHandler` answers a 401 with that message and nothing else, so a developer has nowhere else to read it.

**The `Mock-Key-Id` branch must stay ahead of the "unsigned means Anonymous" rule.** A local test harness names its actor with that header and *no* `Signature` header at all, so an early unsigned guard silently kills local signed-request testing. A real signature that verifies still wins over the header, and off a local hostname the header grants nothing (BUG-51).

## A StreamSource's Status is written by the queue's hooks, never inside the sync

`consumer.WithSession` runs a task inside `factory.WithTransaction`, and a handler that returns an error **aborts that transaction** — so a `FAILURE` status written by `StreamSource.Sync` would be rolled back along with the attempt that produced it, and the record would keep reading `SUCCESS` from last week. The three status writes therefore live in `Consumer.OnSuccess`/`OnError`/`OnFailure` ([consumer/syncStreamSource.go](../consumer/syncStreamSource.go)), which run after the transaction has already settled and open their own session.

`Sync` itself writes only `Version`, `ContentHash`, and `LastSynced`, and only on paths that succeeded. That split is also why `LOADING` is written by `StreamSource.Save` rather than by `Sync`: a save runs in a request transaction that commits, so a human who pressed **Sync Now** sees it immediately.

## Every ping moves "Last Checked", and only a hook can record a failed one

`LastSynced` is what the settings screen labels **Last Checked**, and the rule is that every attempt moves it — a webhook ping and the **Sync Now** button alike, whether the attempt then worked or not. `markChecked` is the one writer; `Save`, `SetStatusLoading`, `SetStatusSuccess`, `SetStatusFailure`, and `SetStatusMessage` all call it.

The three status methods are load-bearing and look redundant. `Sync` stamps the record itself, but only its three SAVING exits reach a write: the four error exits return before one, `consumer.WithSession` aborts the transaction, and the stamp is rolled back with everything else. The lifecycle hook that fires afterwards reloads the record from the database, so whatever `Sync` held in memory is gone. That hook is the **only** place a failed attempt can record that it happened.

The webhook path makes this matter more than the button does. `SyncByWebhookToken` writes nothing at all — it just queues one task per matching record — so unlike **Sync Now**, which stamps the record through `Save` before the worker starts, a webhook leaves no trace until the worker's outcome is written. Before this rule a failed webhook sync showed `Failed` beside a Last Checked from days earlier, which reads as a webhook that never arrived rather than one that arrived and failed.

`consumeSafely` reaches no hook (see [../consumer/AGENTS.md](../consumer/AGENTS.md)), so a task that PANICS still records nothing. That gap is unchanged and is the reason a handler whose status a human reads must return a `queue.Result` instead of panicking.

## `Sync` saves through `service.save`, never through `StreamSource.Save`

`StreamSource.Save` publishes a sync task — that is how a new record gets its first content and how a corrected URL is retried, since nothing polls. A sync that saved its own bookkeeping through it would queue another sync on every run, forever. `WithSignature` does not stop this, because the task that is running has already left the queue by the time its handler saves. `saveSyncState` and every `SetStatus*` method therefore write through the package-private `service.save`, and that is load-bearing rather than an optimization.

The two are easy to confuse because only one letter differs. **`StreamSource.Save`** (exported) validates, stamps `LOADING`, and queues a synchronization. **`service.save`** (private) writes the row and publishes the SSE nudge, and does neither of the other two. Every write goes through one of them; nothing calls `service.collection(session).Save` any more.

## `StreamSource.Version` is an ETag, and an empty one can never end a sync

The field holds the `ETag` from the last successful sync, and its only purpose is to be sent back as `If-None-Match` so that an unchanged source can answer `304` with no body. `Sync` and the HTTPS adapter are its only readers. It is a bandwidth optimization, not a correctness guard — `ContentHash` is what stops `Stream.Save` running for unchanged bytes, on every forge, including the two that send no validator.

Two of the seven forges surveyed offer no validator at all: cgit ignores `If-None-Match`, and SourceHut sends no `ETag`. They answer `""` every time, so `version == source.Version` is trivially true for them, and treating that as "nothing changed" would freeze those sources at whatever they held on the first run, silently and forever. The guard is `(version != "") && (version == source.Version)`.

## The webhook token is deliberately not unique, and an empty one would select rather than authorize

One token belongs to many `StreamSource` records, so one ping from a repository refreshes every page sourced from it. That makes a permissive match worse than it looks: an empty token would match every record whose `webhookToken` was never set and start a sync on each, which is fan-out triggered by an unauthenticated caller rather than an authorization check that merely passed. `RangeByWebhookToken` refuses anything shorter than `model.StreamSourceWebhookTokenMinLength` before it queries, and `Save` enforces the same minimum, because the field is editable by hand.

The token is looked up, not compared, so there is no constant-time comparison to make here. What protects it is that [handler/streamSource.go](../handler/streamSource.go) answers `202` for every outcome — a token too short to look up, an unknown token, a known token matching nothing, and a known token matching forty records. A varying answer would confirm a guessed token and then count the pages behind it.

## Saving a StreamSource reaches the network

There is no step, and no flag, that triggers a synchronization: **`StreamSource.Save` queues one, every time.** Nothing polls, so a save is the only moment a human tells Emissary this record is worth reading, and a redundant one costs a single conditional GET that answers `304` — `Version` matches, `Fetch` never runs, and `Stream.Save` never fires. **Sync Now** is therefore `{do:"with-stream-source", steps:[{do:"save"}]}`.

Two consequences to keep in mind. An unrelated edit made while the source is unreachable will walk the record to `FAILURE`, which the next webhook corrects. And a caller that saves MANY records in a loop fans out one outbound fetch per record with nothing at the call site saying so — `step_Sort.go` reaches `Save` through `ObjectSave` and is one hjson line away from being such a caller.

Bookkeeping writes avoid all of this by design. `saveSyncState` and every `SetStatus*` method go through `service.save`, which is what stops a running sync from queueing another one. `WithSignature` would NOT save you there: the task that is running has already left the queue by the time its handler saves.

## Two task names synchronize a StreamSource, and four places must know both

`TaskSyncStreamSource` is the webhook's background fan-out at priority 256. `TaskSyncStreamSourceNow` is the **Sync Now** button at 16. They run the same handler and report through the same lifecycle hooks; only the priority and the signature differ.

The signature is the reason there are two names rather than one publish option. turbine's `allowImmediate` refuses to run **any** signed task from memory, at any priority, because signature dedup needs a stored row to check against — so a signed task waits for the storage poller, which sleeps a minute when the queue is idle. `PublishSyncTaskNow` therefore omits the signature, and `PublishSyncTask` keeps it.

What that costs: two quick presses of **Sync Now** queue two syncs, and one may overlap a webhook's background sync of the same record. A repeat is one conditional `GET` answering `304`, so the usual case is free. The case that is not free is a caller that saves many records at once — `step_Sort.go` reaches `Save` through `ObjectSave` — which now fans out immediate fetches instead of deduplicated background ones.

Four registrations must accept both names: the dispatch switch in `consumer.go`, and all three lifecycle hooks. Use `service.IsSyncStreamSourceTask` rather than comparing a name by hand — a hook that knew only one name would silently stop recording status for the other path, and nothing reports a hook that declined a task. The priority table is the one place that deliberately tells them apart.

## `with-stream-source` needs two registrations, and neither fails to compile

`model.StreamSource` must implement `model.AccessLister`, and `*model.StreamSource` must have a case in `Factory.ModelService`. Miss either and `NewModel` returns nil and the settings screen 500s the first time it is opened.

[build/step_WithStreamSource.go](../build/step_WithStreamSource.go) builds its load target with `NewStreamSource`, the same as `Range` and `ObjectLoad`, so a field the model gains later arrives with its default rather than a zero value. That is safe here only because `Save` refuses a record whose webhook token is under 16 characters, so every stored record names `Config["webhookToken"]` and the decode overwrites the minted one. A second `Config` key that is not always written would leak out of the constructor — see the decode rule in [../model/AGENTS.md](../model/AGENTS.md).

## A StreamSource change is announced on its STREAM's SSE channel

The settings screen shows `Status` and the time of the last check, and a synchronization finishes in the background minutes after **Sync Now** returns — so without a nudge those two fields sit stale until somebody reloads. `service.save` publishes `realtime.TopicStreamSourceUpdated` on every write, and `Delete` publishes it too, so the page can fall back to its "no source yet" state.

`saveSyncState` is the one exception, and it is deliberate: every path through `Sync` is followed by a lifecycle hook that writes `Status` and `LastSynced` — the same two fields the screen shows — a few hundred milliseconds later. Nudging from both made the settings screen redraw twice for one synchronization. `Delete` is the other non-nudging write; its action ends with `refresh-page`.

**Never put a modifier on an `sse:` trigger in a template.** htmx's `hx-trigger` parser handles `sse:` in its own branch and pushes the spec without parsing modifiers, so `delay`, `throttle`, `from` and `once` are ignored there — and the unparsed tokens halt the parser, so every spec after the next comma is silently dropped. `hx-trigger="sse:X delay:300ms, refreshPage from:window"` therefore registers **only** the SSE trigger, with no debounce, and deletes the `refreshPage` listener that the properties modal depends on. No error is raised. A comma straight after the event name is fine — `"sse:X, refreshPage from:window"` registers both — it is the modifier that halts the parse. There is also no other client-side brake to reach for: core's `processSSETrigger` calls `issueAjaxRequest` from the `EventSource` listener directly, skipping the queue, the throttle, and `hx-sync`. De-duplicate by writing the record fewer times, or by swapping a smaller region — never by debouncing the client.

The message is addressed by the **StreamID**, not the StreamSourceID. A `StreamSource` has no page and no SSE route of its own, and a Stream has at most one source, so the Stream's channel is the one a browser can already subscribe to (`/:stream/sse/stream-source-updated`). Putting the nudge on `service.save` rather than on the four `SetStatus*` methods is what makes it impossible to add a fifth status writer that the screen never hears about.

## Publishing to followers happens in two phases, and the second reads different fields than it sends

`Outbox.Publish` does only database work, inside the caller's transaction. It saves the OutboxMessage, stamps the activity's `id` and `actor`, and queues an `Outbox-Publish` task. `Outbox.Deliver` runs that task after the transaction commits, on its own session, so no signed HTTP request ever runs inside an open transaction. The design, including why Emissary owns this task rather than using hannibal's sender fan-out, is in [POST-COMMIT-FEDERATION.md](../../emissary-specs/projects/_done/POST-COMMIT-FEDERATION.md) (F1, F2).

`Deliver` builds the recipient list from the activity's **full** addressing, because blind recipients (`bto`, `bcc`) must still receive the post. It then **sends** a copy with `bto` and `bcc` removed, so no recipient sees who else was blind-copied. Passing the original to a delivery call leaks the blind lists to every follower, which happened until §4.2 of HEAP-ALLOCATIONS.md; `TestDeliver_StripsBlindRecipients` guards it.

That stripped copy is serialized to JSON **once per fan-out**, and every delivery task carries the same string as its `body` argument, which hannibal's sender POSTs as-is. Carrying the whole activity map per follower instead cost 233 allocations per follower once the task went through storage. Because an older Emissary reads only the `activity` map, it would POST `{}` for these tasks and drop them, so every process sharing a database must be upgraded together, and a rollback must wait until no delivery tasks are queued. See hannibal's AGENTS.md for the handler side.

Two more rules from [COLLECTIONS-REDESIGN.md](../../emissary-specs/projects/_done/COLLECTIONS-REDESIGN.md) are easy to break. `UndoActivity` embeds the original activity instead of linking it, because the record behind that link is often already deleted (D7). An addressee carries only a `ProfileURL`, so any check on a recipient's host must fall back from `InboxURL` to `ProfileURL`, or author-only delivery silently breaks on a localhost domain (D8).
