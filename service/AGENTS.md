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

## Domain bootstrap is one transaction, and the invariant is what matters

`Domain.Start()` delegates to `bootstrap(session)`, which wraps the domain-record write and `createOwner` in a single transaction. That establishes **domain record exists if and only if an owner exists** (when `CreateOwner` is set), so a failed first boot writes nothing and the next boot retries cleanly. Before this, the owner was a separate non-transactional write gated on "domain record not found": any failure stranded a domain record with no owner and the gate never re-ran, which locked every demo and fresh instance out. `persist()` is the write-only path used inside the transaction; the in-memory domain cache is published only **after** commit. Do not collapse `persist` back into `Save`.

Three decisions here look like defects and are not. The `admin`/`admin` default password is set **only** under `IsLocalhost()`; that gate is the thing keeping a known credential off public hosts. `config.Owner` has no password field by design — a non-localhost owner signs in first through an emailed reset link, or the operator gets a loud warning pointing at the setup console. And `newOwnerFromConfig` falls back from a blank email to `admin@<hostname>` because `User.Save` requires a non-empty address.

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

## A failed Actor load becomes a Following status in one place

`Following.SetStatusPollError` ([following_pollError.go](following_pollError.go)) owns the whole meaning of a failed Actor load: `GONE` on a `410`, `FAILURE` otherwise (escalating to `PAUSED` through `SetStatusPollFailure`), and the sentence the owner reads. Callers pass the raw error and never pick a status or write the text themselves. Before this, the poller chose `GONE` while the service chose `PAUSED`, so one state machine lived in two packages, and the connect path showed a generic message for the same failure the poller described precisely. A `429` must never reach it: a rate limit is the host's throttle, so callers requeue it first, and the method would otherwise record it as the record's own failure.

Every Following setter that takes a caller's message bounds it to the 1024-byte `statusMessage` schema itself, with `truncateStatusMessage`, the same way `StreamSource.SetStatusFailure` does. A message that quotes a remote root message has no bound of its own, so a caller should never have to remember one.
